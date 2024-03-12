package ase

import (
	"fmt"
)

// RespHandler to handle each response reading from ASE server
type RespHandler interface {
	Handle(data *Resp) error
	Destroy() error
}

type defaultHandler struct {
}

func (d *defaultHandler) Handle(data *Resp) error {
	fmt.Printf("%+v\n", data)
	return nil
}

func (d *defaultHandler) Destroy() error {
	return nil
}
