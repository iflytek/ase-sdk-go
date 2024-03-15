package ase

import (
	"encoding/json"
)

type Decoder interface {
	Decode(data []byte) (*Resp, error)
}

type defaultDecoder struct {
}

func (d *defaultDecoder) Decode(data []byte) (*Resp, error) {
	var resp Resp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
