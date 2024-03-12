# ase-sdk-go

## Installation
```go
go get github.com/iflytek/ase-sdk-go
```

## Usage
```go
package main

import (
	"fmt"

	"github.com/iflytek/ase-sdk-go"
)

func main() {
	cli, err := ase.NewClient(
		"appid",
		"apikey",
		"secret",
		"https://example.com",
		"/example",
	)
	if err != nil {
		panic(err)
	}

	resp, err := cli.Once(map[string]interface{}{
		"header": map[string]interface{}{},     // 根据具体业务参数填充
		"parameter": map[string]interface{}{},  // 根据具体业务参数填充
		"payload": map[string]interface{}{},    // 根据具体业务参数填充
	})
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("response: %+v\n", resp)
}
```
