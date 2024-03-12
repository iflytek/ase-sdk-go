package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/iflytek/ase-sdk-go"
)

const (
	endpoint = "https://itrans.xf-yun.com"
	uri      = "/v1/its"
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
	fmt.Println("翻译结果: ", Trans("你好", "cn", "en"))
}

// Trans 翻译text为指定语言的文本
func Trans(raw, from, to string) string {
	text := base64.StdEncoding.EncodeToString([]byte(raw))

	data := map[string]interface{}{
		"header": map[string]interface{}{
			"app_id": appid,
			"status": 3,     // 请求状态 取固定值3
			"res_id": "123", // 个性化资源id, 由调用个性化上传接口时指定， 非必填
		},
		"parameter": map[string]interface{}{
			"its": map[string]interface{}{
				"from":   from,
				"to":     to,
				"result": map[string]interface{}{},
				"domain": "common", // 领域参数
			},
		},
		"payload": map[string]interface{}{
			"input_data": map[string]interface{}{
				"status": 3, // 数据状态， 取固定值3
				"text":   text,
			},
		},
	}

	cli, err := ase.NewClient(
		appid,
		apikey,
		secret,
		endpoint,
		uri,
		ase.WithDecoder(&transDecoder{}),
		ase.WithOnceTimeout(time.Second*5),
		ase.WithOnceRetryCount(3),
	)
	if err != nil {
		panic(err)
	}

	resp, err := cli.Once(data)
	if err != nil {
		panic(err)
	}

	return resp.Payload.(Payload).DecodedText.TransResult.Dst
}
