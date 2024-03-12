package ase

const (
	StatusFirstFrame = 0
	StatusContinue   = 1
	StatusLastFrame  = 2
)

type Request struct {
	Header    map[string]interface{} `json:"header"`
	Parameter map[string]interface{} `json:"parameter"`
	Payload   map[string]interface{} `json:"payload"`
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
