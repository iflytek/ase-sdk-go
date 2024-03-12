package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/iflytek/ase-sdk-go"
	"github.com/pkg/errors"
)

const (
	file = "./test.en.txt.pcm"

	endpoint = "ws://cn-huadong-1.xf-yun.com"
	uri      = "/v1/private/s501d1f86"
	// endpoint = "ws://172.31.103.99:8888" // 测试环境

	frameSize = 1024 //每一帧的音频大小
)

var (
	appid     string
	apikey    string
	apiSecret string
)

func main() {
	appid = os.Getenv("APPID")
	apikey = os.Getenv("APIKEY")
	apiSecret = os.Getenv("API_SECRET")

	h := NewHandler()
	defer func() {
		_ = h.Destroy()
	}()

	cli, err := ase.NewClient(appid, apikey, apiSecret, endpoint, uri, ase.WithDecoder(&istDecoder{}))
	if err != nil {
		panic(err)
	}

	data := make(chan ase.Request)
	go func() {
		mockSend(data)
	}()

	errChan, done := cli.Stream(data, h.Handle)

	select {
	case err = <-errChan:
		fmt.Printf("stream err: %+v\n", err)
		return
	case <-done:
		fmt.Println("stream done")
		return
	}
}

func mockSend(data chan ase.Request) {
	audioBytes, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	var status int
	chunks := doChunk(audioBytes, frameSize)

	for i := 0; i < len(chunks); i++ {
		if i == 0 {
			status = ase.StatusFirstFrame
		} else if i == len(chunks)-1 {
			status = ase.StatusLastFrame
		} else {
			status = ase.StatusContinue
		}

		frame := input(chunks[i], status)

		switch status {
		case ase.StatusFirstFrame:
			fmt.Println("======Sending First Frame=====")
			data <- *frame
		case ase.StatusContinue:
			data <- *frame
		case ase.StatusLastFrame:
			fmt.Println("======Sending Last Frame=====")
			data <- *frame
			close(data)
		}
		//模拟音频采样间隔
		//time.Sleep(interval)
	}
}

func input(data []byte, status int) *ase.Request {
	var frameData *ase.Request
	switch status {
	case ase.StatusFirstFrame:
		frameData = &ase.Request{
			Header: map[string]interface{}{
				"app_id": appid, //appid 必须带上，只需第一帧发送
				"status": status,
			},
			Parameter: map[string]interface{}{
				"iat": map[string]interface{}{
					//"dwa":      "wpgs",
					"language": "en_us",
					"result": map[string]interface{}{
						"encoding": "utf8",
						"compress": "raw",
						"format":   "json",
					},
				},
			},
			Payload: map[string]interface{}{
				"audio": map[string]interface{}{
					"encoding":    "raw",
					"sample_rate": 16000,
					"channels":    1,
					"bit_depth":   16,
					"status":      status,
					"seq":         0,
					"audio":       base64.StdEncoding.EncodeToString(data),
					"frame_size":  frameSize,
				},
			},
		}
	case ase.StatusContinue:
		frameData = &ase.Request{
			Header: map[string]interface{}{
				"app_id": appid, //appid 必须带上，只需第一帧发送
				"status": status,
			},
			Payload: map[string]interface{}{
				"audio": map[string]interface{}{
					"encoding":    "raw",
					"sample_rate": 16000,
					"channels":    1,
					"bit_depth":   16,
					"status":      status,
					"seq":         0,
					"audio":       base64.StdEncoding.EncodeToString(data),
					"frame_size":  frameSize,
				},
			},
		}
	case ase.StatusLastFrame:
		frameData = &ase.Request{
			Header: map[string]interface{}{
				"app_id": appid, //appid 必须带上，只需第一帧发送
				"status": status,
			},
			Payload: map[string]interface{}{
				"audio": map[string]interface{}{
					"encoding":    "raw",
					"sample_rate": 16000,
					"channels":    1,
					"bit_depth":   16,
					"status":      status,
					"seq":         0,
					"audio":       base64.StdEncoding.EncodeToString(data),
					"frame_size":  frameSize,
				},
			},
		}
	}

	return frameData
}

func doChunk(data []byte, size int) (res [][]byte) {
	for i := 0; i < len(data); i += size {
		end := i + size
		if end > len(data) {
			end = len(data)
		}

		res = append(res, data[i:end])
	}

	return
}

type IstResp struct {
	Header  *ase.Header `json:"header"`
	Payload *Payload    `json:"payload"`
}

type Payload struct {
	Result struct {
		Compress       string `json:"compress"`
		Encoding       string `json:"encoding"`
		Format         string `json:"format"`
		Seq            int    `json:"seq"`
		Status         int    `json:"status"`
		Text           string `json:"text"`
		StructuredText Text   `json:"structuredText"`
	} `json:"result"`
}

type Text struct {
	Sn int  `json:"sn"`
	Ls bool `json:"ls"`
	Bg int  `json:"bg"`
	Ed int  `json:"ed"`
	Ws []struct {
		Bg int `json:"bg"`
		Cw []struct {
			Sc int    `json:"sc"`
			W  string `json:"w"`
		} `json:"cw"`
	} `json:"ws"`
}

type istDecoder struct {
}

func (d *istDecoder) Decode(data []byte) (*ase.Resp, error) {
	cus := &IstResp{}
	if err := json.Unmarshal(data, cus); err != nil {
		return nil, err
	}

	if cus.Payload == nil {
		return &ase.Resp{
			Header: cus.Header,
		}, nil
	}

	decodedText, err := base64.StdEncoding.DecodeString(cus.Payload.Result.Text)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode text")
	}

	if err = json.Unmarshal(decodedText, &cus.Payload.Result.StructuredText); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal text")
	}

	return &ase.Resp{
		Header:  cus.Header,
		Payload: cus.Payload,
	}, nil
}

type handler struct {
	bodyFile *os.File
}

func NewHandler() ase.RespHandler {
	respFile, _ := os.Create("./resp.body.txt")

	return &handler{
		bodyFile: respFile,
	}
}

func (h *handler) Handle(data *ase.Resp) (err error) {
	var b []byte
	b, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}

	if _, err = h.bodyFile.Write(b); err != nil {
		return
	}
	if _, err = h.bodyFile.WriteString("\n"); err != nil {
		return
	}

	return nil
}

func (h *handler) Destroy() error {
	return h.bodyFile.Close()
}
