package ase

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
)

type ASE interface {
	// Once send a http request to ASE server, and return the response
	Once(data *Request) (body []byte, err error)
	// Receive data from ASE server in websockets
	Receive() (body []byte, err error)
	// Send data to ASE server in websockets
	Send(data *Request) error
	// Destroy resources
	Destroy() error
}

type Sender func()

type client struct {
	appid, apikey, apiSecret   string
	host                       string // eg: iflytek.com
	tls                        bool
	uri                        string           // eg: /ase/v1/ping
	signAlg                    func() hash.Hash // hash algorithm using for signature
	signedHttpURL, signedWsURL string

	*onceCaller
	*streamCaller
}

// NewClient create a new client to ASE server.
// endpoint: eg: https://iflytek.com
// uri: eg: /ase/v1/ping
// opts: eg: WithOnceTimeout(time.Second), WithOnceRetryCount(3)
func NewClient(appid, apikey, apiSecret, host, uri string, opts ...Option) (ASE, error) {
	c := &client{
		appid:      appid,
		apikey:     apikey,
		apiSecret:  apiSecret,
		host:       host,
		uri:        uri,
		onceCaller: &onceCaller{cli: resty.New()},
		streamCaller: &streamCaller{
			conn:             nil,
			once:             sync.Once{},
			handshakeTimeout: 0,
			readTimeout:      0,
			writeTimeout:     0,
			onceErr:          nil,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.signAlg == nil {
		c.signAlg = sha256.New
	}

	c.signedWsURL = c.buildSignedURL(c.host, c.uri, http.MethodGet)
	c.signedHttpURL = c.buildSignedURL(c.host, c.uri, http.MethodPost)

	return c, nil
}

type Option func(*client)

func WithTLS() Option {
	return func(c *client) {
		c.tls = true
	}
}

func WithOnceTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.onceCaller.cli.SetTimeout(timeout)
	}
}

func WithOnceRetryCount(count int) Option {
	return func(c *client) {
		c.onceCaller.cli.SetRetryCount(count)
	}
}

func WithStreamHandshakeTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.handshakeTimeout = timeout
	}
}

func WithStreamReadTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.readTimeout = timeout
	}
}

func WithStreamWriteTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.writeTimeout = timeout
	}
}

func WithSignAlgorithm(alg func() hash.Hash) Option {
	return func(c *client) {
		c.signAlg = alg
	}
}

func WithStreamConnTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.streamCaller.connTimeout = timeout
	}
}

type onceCaller struct {
	cli *resty.Client
}

type streamCaller struct {
	conn             *websocket.Conn
	connTimeout      time.Duration // 连接保活时间, 默认无
	handshakeTimeout time.Duration // 握手超时时间, 默认无
	readTimeout      time.Duration
	writeTimeout     time.Duration

	once    sync.Once
	onceErr error
}

func (c *client) Once(data *Request) (resp []byte, err error) {
	var (
		res *resty.Response
	)

	res, err = c.cli.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Post(c.signedHttpURL)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("http_code: %d, http_msg: %s, body: %s", res.StatusCode(), res.Status(), string(res.Body()))
	}

	return res.Body(), nil
}

func (c *client) Receive() (msg []byte, err error) {
	c.once.Do(func() {
		c.onceErr = c.initWebsocketConn(c.signedWsURL)
	})

	if c.onceErr != nil {
		return nil, c.onceErr
	}

	if c.readTimeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	_, msg, err = c.conn.ReadMessage()
	return
}

func (c *client) Send(v *Request) (err error) {
	c.once.Do(func() {
		c.onceErr = c.initWebsocketConn(c.signedWsURL)
	})

	if c.onceErr != nil {
		return c.onceErr
	}

	if c.writeTimeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	return c.conn.WriteJSON(v)
}

func (c *client) initWebsocketConn(url string) error {
	d := websocket.Dialer{
		NetDial:           nil,
		NetDialContext:    nil,
		NetDialTLSContext: nil,
		Proxy:             nil,
		TLSClientConfig:   nil,
		HandshakeTimeout:  c.handshakeTimeout,
		ReadBufferSize:    0,
		WriteBufferSize:   0,
		WriteBufferPool:   nil,
		Subprotocols:      nil,
		EnableCompression: false,
		Jar:               nil,
	}

	var (
		err  error
		resp *http.Response
	)

	c.conn, resp, err = d.Dial(url, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http_code: %d, http_msg: %s, body: %s", resp.StatusCode, resp.Status, string(b))
	}

	if c.connTimeout > 0 {
		go func() {
			tm := time.NewTimer(c.connTimeout)
			defer tm.Stop()

			<-tm.C
			_ = c.conn.Close()
		}()
	}

	return nil
}

func (c *client) Destroy() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// buildSignedURL 创建带有签名的url
// @endpoint such as ws://10.1.87.70:80
// @uri such as /v1/ping
func (c *client) buildSignedURL(host, uri, method string) string {
	// 签名时间
	now := time.Now().UTC().Format(time.RFC1123)

	// 待签名字符串
	signText := fmt.Sprintf("host: %s\ndate: %s\n%s %s HTTP/1.1", host, now, method, uri)

	// 签名结果
	signature := NewSigner(c.apiSecret, c.signAlg).Sign([]byte(signText))

	//构建请求参数 此时不需要urlencoding
	urls := fmt.Sprintf("api_key=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", c.apikey,
		"hmac-sha256", "host date request-line", signature)
	// 将请求参数使用base64编码
	urls = base64.StdEncoding.EncodeToString([]byte(urls))

	v := url.Values{}
	v.Add("host", host)
	v.Add("date", now)
	v.Add("authorization", urls)

	return scheme(method, c.tls) + host + uri + "?" + v.Encode()
}

func scheme(method string, tls bool) string {
	if method == http.MethodGet {
		if tls {
			return "wss://"
		}
		return "ws://"
	}

	if tls {
		return "https://"
	}

	return "http://"
}
