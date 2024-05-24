package ase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

func (c *client) buildAIaaSHeader(body []byte) (header map[string]string) {
	header = make(map[string]string)
	//date必须是utc时区，且不能和服务器时间相差300s
	currentTime := time.Now().UTC().Format(time.RFC1123)
	//对body进行sha256签名,生成digest头部，POST请求必须对body验证
	digest := "SHA-256=" + signBody(body)
	//根据请求头部内容，生成签名
	sign := generateSignature(c.host, currentTime, http.MethodPost, c.uri, "HTTP/1.1", digest, c.apiSecret)
	//组装Authorization头部
	authHeader := fmt.Sprintf(`hmac api_key="%s", algorithm="%s", headers="host date request-line digest", signature="%s"`, c.apikey, "hmac-sha256", sign)

	header["Content-Type"] = "application/json"
	header["Host"] = c.host
	header["Date"] = currentTime
	header["Digest"] = digest
	header["Authorization"] = authHeader

	return
}

func signBody(data []byte) string {
	sha := sha256.New()
	sha.Write(data)
	encodeData := sha.Sum(nil)
	return base64.StdEncoding.EncodeToString(encodeData)
}

func generateSignature(host, date, httpMethod, requestUri, httpProto, digest string, secret string) string {

	//不是request-line的话，则以 header名称,后跟ASCII冒号:和ASCII空格，再附加header值
	var signatureStr string
	if len(host) != 0 {
		signatureStr = "host: " + host + "\n"
	}
	signatureStr += "date: " + date + "\n"

	//如果是request-line的话，则以 http_method request_uri http_proto
	signatureStr += httpMethod + " " + requestUri + " " + httpProto + "\n"
	signatureStr += "digest: " + digest
	return hmacsign(signatureStr, secret)
}

func hmacsign(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	encodeData := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(encodeData)
}
