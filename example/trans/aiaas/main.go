package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/iflytek/ase-sdk-go"
)

var (
	appid  string
	apikey string
	secret string
	host   string
	uri    string
)

func main() {
	appid = os.Getenv("APPID")
	apikey = os.Getenv("APIKEY")
	secret = os.Getenv("API_SECRET")
	host = os.Getenv("HOST")
	uri = os.Getenv("URI")

	Trans("你好啊, 自由的世界!", "cn", "en")
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
	)
	if err != nil {
		panic(err)
	}

	resp, err := cli.OnceAIaaS(&ase.AIaaSRequest{
		Common: map[string]interface{}{
			"app_id": appid,
			"status": ase.StatusForOnce,
		},
		Business: map[string]interface{}{
			"from":   from,
			"to":     to,
			"domain": "common",
		},
		Data: map[string]interface{}{
			"text": text,
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", string(resp))
}
