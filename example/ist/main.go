package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	"github.com/iflytek/ase-sdk-go"
)

const (
	file = "./test.en.txt.pcm"

	host = "cn-huadong-1.xf-yun.com"
	uri  = "/v1/private/se671b848"

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

	cli, err := ase.NewClient(appid, apikey, apiSecret, host, uri)
	if err != nil {
		panic(err)
	}

	audioBytes, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	var status int
	chunks := doChunk(audioBytes, frameSize)
	for i, chunk := range chunks {
		if i == 0 {
			status = ase.StatusFirstFrame
		} else if i == len(chunks)-1 {
			status = ase.StatusLastFrame
		} else {
			status = ase.StatusContinue
		}

		req := new(ase.Request)
		req.SetHeaders(&ase.RequestHeader{
			AppId:  appid,
			Status: status,
		})
		if status == ase.StatusFirstFrame {
			req.SetParameters(map[string]interface{}{
				"iat": map[string]interface{}{
					//"dwa":      "wpgs",
					"language": "en_us",
					"result": map[string]interface{}{
						"encoding": "utf8",
						"compress": "raw",
						"format":   "json",
					},
				},
			})
		}
		req.SetPayloads(map[string]interface{}{
			"audio": map[string]interface{}{
				"encoding":    "raw",
				"sample_rate": 16000,
				"channels":    1,
				"bit_depth":   16,
				"status":      status,
				"seq":         0,
				"audio":       base64.StdEncoding.EncodeToString(chunk),
				"frame_size":  frameSize,
			},
		})

		if err = cli.Send(req); err != nil {
			fmt.Printf("send error: %+v\n", err)
			return
		}

		if status == ase.StatusFirstFrame {
			wg.Add(1)
			go read(cli, &wg)
		}
	}

	wg.Wait()
}

func read(cli ase.ASE, wg *sync.WaitGroup) {
	h := NewHandler()
	defer wg.Done()
	defer h.Destroy()

	for {
		msg, err := cli.Receive()
		if err != nil {
			fmt.Printf("receive error: %+v\n", err)
			return
		}

		if len(msg) == 0 {
			continue
		}

		resp, err := new(istDecoder).Decode(msg)
		if err != nil {
			fmt.Printf("failed to decode: %+v\n", err)
			return
		}

		if err = h.Handle(resp); err != nil {
			fmt.Printf("handle error: %+v\n", err)
			return
		}

		if resp.Header.Status == ase.StatusLastFrame {
			_ = cli.Destroy()
			return
		}
	}
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
