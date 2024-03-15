package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/iflytek/ase-sdk-go"
	"github.com/pkg/errors"
	"os"
)

type IstResp struct {
	Header  *ase.Header `json:"header"`
	Payload *Payload    `json:"payload"`
}

type Payload struct {
	Result struct {
		Compress       string `json:"compress"`
		Encoding       string `json:"encoding"`
		Format         string `json:"format"`
		Seq            int    `json:"seq"`
		Status         int    `json:"status"`
		Text           string `json:"text"`
		StructuredText Text   `json:"structuredText"`
	} `json:"result"`
}

type Text struct {
	Sn int  `json:"sn"`
	Ls bool `json:"ls"`
	Bg int  `json:"bg"`
	Ed int  `json:"ed"`
	Ws []struct {
		Bg int `json:"bg"`
		Cw []struct {
			Sc int    `json:"sc"`
			W  string `json:"w"`
		} `json:"cw"`
	} `json:"ws"`
}

type istDecoder struct {
}

func (d *istDecoder) Decode(data []byte) (*ase.Resp, error) {
	cus := &IstResp{}
	if err := json.Unmarshal(data, cus); err != nil {
		return nil, err
	}

	if cus.Payload == nil {
		return &ase.Resp{
			Header: cus.Header,
		}, nil
	}

	decodedText, err := base64.StdEncoding.DecodeString(cus.Payload.Result.Text)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode text")
	}

	if err = json.Unmarshal(decodedText, &cus.Payload.Result.StructuredText); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal text")
	}

	return &ase.Resp{
		Header:  cus.Header,
		Payload: cus.Payload,
	}, nil
}

type Handler struct {
	bodyFile *os.File
}

func NewHandler() *Handler {
	respFile, _ := os.Create("./resp.body.txt")

	return &Handler{
		bodyFile: respFile,
	}
}

func (h *Handler) Handle(data *ase.Resp) (err error) {
	var b []byte
	b, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}

	if _, err = h.bodyFile.Write(b); err != nil {
		return
	}
	if _, err = h.bodyFile.WriteString("\n"); err != nil {
		return
	}

	return nil
}

func (h *Handler) Destroy() error {
	return h.bodyFile.Close()
}
