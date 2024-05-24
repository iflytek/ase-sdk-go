package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/iflytek/ase-sdk-go"
)

const (
	file      = "./example/ist/test.en.txt.pcm"
	frameSize = 1024 //每一帧的音频大小
	interval  = 40
)

var (
	appid     string
	apikey    string
	apiSecret string
	host      string
	uri       string
)

func main() {
	appid = os.Getenv("APPID")
	apikey = os.Getenv("APIKEY")
	apiSecret = os.Getenv("API_SECRET")
	host = os.Getenv("HOST")
	uri = os.Getenv("URI")

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

		if err = cli.SendAIaaS(&ase.AIaaSRequest{
			Common: map[string]interface{}{
				"app_id": appid,
			},
			Business: map[string]interface{}{
				//"dwa":      "wpgs",
				"language": "en_us",
				"domain":   "ist_mul_sp",
				"accent":   "mandarin",
			},
			Data: map[string]interface{}{
				"encoding": "raw",
				"status":   status,
				"format":   "audio/L16;rate=16000",
				"audio":    base64.StdEncoding.EncodeToString(chunk),
			},
		}); err != nil {
			fmt.Printf("send error: %+v\n", err)
			return
		}

		if status == ase.StatusFirstFrame {
			wg.Add(1)
			go read(cli, &wg)
		}

		time.Sleep(time.Millisecond * interval)
	}

	wg.Wait()
}

func read(cli ase.ASE, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		msg, err := cli.Receive()
		if err != nil {
			fmt.Printf("receive error: %+v\n", err)
			return
		}

		if len(msg) == 0 {
			continue
		}

		fmt.Printf("receive: %s\n", string(msg))

		var j Response
		if err = json.Unmarshal(msg, &j); err != nil {
			fmt.Printf("unmarshal response error: %+v\n", err)
			return
		}

		if j.Data.Status == ase.StatusLastFrame {
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

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Sid     string `json:"sid"`
	Data    struct {
		Result struct {
			Sn     int  `json:"sn"`
			Ls     bool `json:"ls"`
			Bg     int  `json:"bg"`
			Ed     int  `json:"ed"`
			SubEnd bool `json:"sub_end"`
			Ws     []struct {
				Bg int `json:"bg"`
				Cw []struct {
					Sc int    `json:"sc"`
					W  string `json:"w"`
				} `json:"cw"`
			} `json:"ws"`
		} `json:"result"`
		Status int `json:"status"`
	} `json:"data"`
	ContextId string `json:"context_id"`
}
