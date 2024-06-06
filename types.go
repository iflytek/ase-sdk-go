package ase

const (
	StatusFirstFrame = 0
	StatusContinue   = 1
	StatusLastFrame  = 2
	StatusForOnce    = 3
)

type Request struct {
	Header    RequestHeader          `json:"header"`
	Parameter map[string]interface{} `json:"parameter,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
}

type TextPayload struct {
	Status int    `json:"status"`
	Text   string `json:"text"`
}

type AudioPayload struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	BitDepth   int    `json:"bit_depth"`
	Status     int    `json:"status"`
	Seq        int    `json:"seq"`
	Audio      string `json:"audio"`
	FrameSize  int    `json:"frame_size"`
}

type ImagePayload struct {
	Status int    `json:"status"`
	Image  string `json:"image"`
}

//type VideoPayload struct {
//	Status int `json:"status"`
//}
//
//type ResourcePayload struct {
//	Status int `json:"status"`
//}

// RequestHeader 平台参数
type RequestHeader map[string]interface{}

func (h RequestHeader) Set(key, value string) {
	h[key] = value
}

func (h RequestHeader) SetAppID(appid string) {
	h["app_id"] = appid
}

func (h RequestHeader) SetStatus(status int) {
	h["status"] = status
}

func (h RequestHeader) SetResID(resId string) {
	h["res_id"] = resId
}

func (h RequestHeader) SetDirectEng(eng string) {
	h["directEngIp"] = eng
}

func (req *Request) SetHeaders(headers RequestHeader) {
	req.Header = headers
}

func (req *Request) SetParameter(key string, value interface{}) {
	if req.Parameter == nil {
		req.Parameter = make(map[string]interface{})
	}
	req.Parameter[key] = value
}

func (req *Request) SetParameters(params map[string]interface{}) {
	req.Parameter = params
}

func (req *Request) SetTextPayload(key string, payload *TextPayload) {
	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
	}
	req.Payload[key] = payload
}

func (req *Request) SetAudioPayload(key string, payload *AudioPayload) {
	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
	}
	req.Payload[key] = payload
}

func (req *Request) SetImagePayload(key string, payload *ImagePayload) {
	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
	}
	req.Payload[key] = payload
}

func (req *Request) SetPayload(key string, value interface{}) {
	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
	}

	req.Payload[key] = value
}

func (req *Request) SetPayloads(payloads map[string]interface{}) {
	req.Payload = payloads
}

type Resp struct {
	Header  *Header     `json:"header"`
	Payload interface{} `json:"payload"`
}

type Header struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Sid     string `json:"sid"`
	Status  int    `json:"status"`
}

type AIaaSRequest struct {
	Common   map[string]interface{} `json:"common"`
	Business map[string]interface{} `json:"business"`
	Data     map[string]interface{} `json:"data"`
}
