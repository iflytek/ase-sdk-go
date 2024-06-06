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
		ase.WithOnceTimeout(time.Second*5),
		ase.WithOnceRetryCount(3),
		ase.WithTLS(),
	)
	if err != nil {
		panic(err)
	}

	headers := ase.RequestHeader{}
	headers.SetAppID(appid)
	headers.SetStatus(ase.StatusForOnce)

	req := new(ase.Request)
	req.SetHeaders(headers)
	req.SetParameters(map[string]interface{}{
		"its": map[string]interface{}{
			"from":   from,
			"to":     to,
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

	res, err := new(transDecoder).Decode(resp)
	if err != nil {
		panic(err)
	}

	fmt.Printf(res.Payload.(Payload).DecodedText.TransResult.Dst)
}
