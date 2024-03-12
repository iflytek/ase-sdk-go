package ase

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"
)

var (
	ErrorConnDeadlined = errors.New("connection deadlined")
)

type ASE interface {
	// Once send a http request to ASE server, and return the response body
	Once(data interface{}) (body *Resp, err error)
	// Stream send/receive requests/responses to/from ASE server in websockets asynchronously, wrapped Write/Read in loop.
	// Streaming write data from data channel to ASE, and streaming read from ASE by providing a response handler.
	Stream(data <-chan Request, respCallback func(resp *Resp) error) (err chan error, done chan struct{})
	// Read data from ASE server in websockets
	Read() (resp *Resp, err error)
	// WriteJSON write data to ASE server in websockets
	WriteJSON(data interface{}) error
	// Destroy resources
	Destroy() error
}

type client struct {
	appid, apikey, apiSecret string
	endpoint                 string           // eg: https://iflytek.com
	uri                      string           // eg: /ase/v1/ping
	signAlg                  func() hash.Hash // hash algorithm using for signature
	signedURL                string

	decoder Decoder

	*onceCaller
	*streamCaller
}

// NewClient create a new client to ASE server.
// endpoint: eg: https://iflytek.com
// uri: eg: /ase/v1/ping
// opts: eg: WithOnceTimeout(time.Second), WithOnceRetryCount(3)
func NewClient(appid, apikey, apiSecret, endpoint, uri string, opts ...Option) (ASE, error) {
	arr := strings.Split(endpoint, "//")
	if len(arr) != 2 {
		return nil, errors.New("endpoint format error")
	}

	if strings.HasPrefix(arr[0], "ws") && strings.HasPrefix(arr[0], "http") {
		return nil, errors.New("endpoint format error")
	}

	c := &client{
		appid:      appid,
		apikey:     apikey,
		apiSecret:  apiSecret,
		endpoint:   endpoint,
		uri:        uri,
		onceCaller: &onceCaller{cli: resty.New()},
		streamCaller: &streamCaller{
			conn:             nil,
			connDead:         make(chan struct{}),
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

	if c.decoder == nil {
		c.decoder = &defaultDecoder{}
	}

	if c.signAlg == nil {
		c.signAlg = sha256.New
	}

	c.signedURL = c.buildSignedURL(c.endpoint, uri)

	return c, nil
}

type Option func(*client)

func WithDecoder(decoder Decoder) Option {
	return func(c *client) {
		c.decoder = decoder
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
	connTimeout      time.Duration // 连接超时时间, 默认无
	connDead         chan struct{}
	handshakeTimeout time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	once             sync.Once
	onceErr          error
}

func (c *client) Once(data interface{}) (body *Resp, err error) {
	var (
		resp *resty.Response
	)

	resp, err = c.cli.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Post(c.signedURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("http_code: %d, http_msg: %s, body: %s", resp.StatusCode(), resp.Status(), string(resp.Body()))
	}

	return c.decoder.Decode(resp.Body())
}

func (c *client) Stream(data <-chan Request, respCallback func(resp *Resp) error) (errChan chan error, done chan struct{}) {
	errChan = make(chan error)
	done = make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.loopRead(respCallback); err != nil {
			errChan <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.loopWrite(data); err != nil {
			errChan <- err
		}
	}()

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	return
}

func (c *client) Read() (resp *Resp, err error) {
	c.once.Do(func() {
		c.onceErr = c.initWebsocketConn(c.signedURL)
	})

	if c.onceErr != nil {
		return nil, c.onError(c.onceErr)
	}

	if c.readTimeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	var msg []byte
	_, msg, err = c.conn.ReadMessage()
	if err != nil {
		return nil, c.onError(err)
	}

	resp, err = c.decoder.Decode(msg)
	return resp, c.onError(err)
}

func (c *client) WriteJSON(v interface{}) (err error) {
	c.once.Do(func() {
		c.onceErr = c.initWebsocketConn(c.signedURL)
	})

	if c.onceErr != nil {
		return c.onError(c.onceErr)
	}

	if c.writeTimeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	return c.onError(c.conn.WriteJSON(v))
}

func (c *client) onError(err error) error {
	if err != nil {
		_ = c.Destroy()
	}
	return err
}

func (c *client) loopRead(respCallback func(resp *Resp) error) error {
	for {
		select {
		case <-c.connDead:
			return ErrorConnDeadlined
		default:
			resp, err := c.Read()
			if err != nil {
				return err
			}

			if err = respCallback(resp); err != nil {
				return err
			}

			if resp.Header.Status == StatusLastFrame {
				_ = c.conn.Close()
				return nil
			}
		}
	}
}

func (c *client) loopWrite(data <-chan Request) error {
	for {
		select {
		case req := <-data:
			status, ok := req.Header["status"]
			if !ok {
				return errors.New("header.status is required")
			}

			if err := c.WriteJSON(req); err != nil {
				return err
			}

			if cast.ToInt(status) == StatusLastFrame {
				return nil
			}
		case <-c.connDead:
			return ErrorConnDeadlined
		}
	}
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
			c.connDead <- struct{}{}
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
func (c *client) buildSignedURL(endpoint, uri string) string {
	// 签名时间
	now := time.Now().UTC().Format(time.RFC1123)

	method := http.MethodPost
	if strings.HasPrefix(endpoint, "ws") {
		method = http.MethodGet
	}

	endpoints := strings.Split(endpoint, "://")

	// 待签名字符串
	signText := fmt.Sprintf("host: %s\ndate: %s\n%s %s HTTP/1.1", endpoints[1], now, method, uri)

	// 签名结果
	signature := NewSigner(c.apiSecret, c.signAlg).Sign([]byte(signText))

	//构建请求参数 此时不需要urlencoding
	urls := fmt.Sprintf("api_key=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", c.apikey,
		"hmac-sha256", "host date request-line", signature)
	// 将请求参数使用base64编码
	urls = base64.StdEncoding.EncodeToString([]byte(urls))

	v := url.Values{}
	v.Add("host", endpoints[1])
	v.Add("date", now)
	v.Add("authorization", urls)
	return endpoint + uri + "?" + v.Encode()
}
