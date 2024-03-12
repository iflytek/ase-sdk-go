package main

import (
	"encoding/base64"
	"encoding/json"

	"github.com/iflytek/ase-sdk-go"
)

type transDecoder struct {
}

func (t *transDecoder) Decode(data []byte) (*ase.Resp, error) {
	tmp := new(Result)
	if err := json.Unmarshal(data, tmp); err != nil {
		return nil, err
	}

	resBs, err := base64.StdEncoding.DecodeString(tmp.Payload.Result.Text)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(resBs, &tmp.Payload.DecodedText)

	return &ase.Resp{
		Header:  tmp.Header,
		Payload: tmp.Payload,
	}, nil
}

type TransResult struct {
	From        string `json:"from"`
	To          string `json:"to"`
	TransResult struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	} `json:"trans_result"`
}

type Result struct {
	Header  *ase.Header `json:"header"`
	Payload Payload     `json:"payload"`
}

type Payload struct {
	Result struct {
		Text string `json:"text"`
	} `json:"result"`
	DecodedText TransResult `json:"decodedText"`
}
