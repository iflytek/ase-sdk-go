package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/iflytek/ase-sdk-go"
)

const (
	host = "itrans.xf-yun.com"
	uri  = "/v1/its"
)

var (
	appid  string
	apikey string
	secret string
)

func main() {
	appid = os.Getenv("APPID")
	apikey = os.Getenv("APIKEY")
	secret = os.Getenv("API_SECRET")

	Trans("你好", "cn", "en")
}

// Trans 翻译text为指定语言的文本
func Trans(raw, from, to string) {
	text := base64.StdEncoding.EncodeToString([]byte(raw))

	cli, err := ase.NewClient(
		appid,
		apikey,
		secret,
		host,
		uri,
		ase.WithDecoder(&transDecoder{}),
		ase.WithOnceTimeout(time.Second*5),
		ase.WithOnceRetryCount(3),
	)
	if err != nil {
		panic(err)
	}

	req := new(ase.Request)
	req.SetHeaders(&ase.RequestHeader{
		AppId:  appid,
		Status: ase.StatusForOnce,
	})
	req.SetParameters(map[string]interface{}{
		"its": map[string]interface{}{
			"from":   "cn",
			"to":     "en",
			"domain": "common",
			"result": map[string]interface{}{},
		},
	})
	req.SetPayloads(map[string]interface{}{
		"input_data": map[string]interface{}{
			"status": ase.StatusForOnce,
			"text":   text,
		},
	})

	resp, err := cli.Once(req)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", string(resp))
}
