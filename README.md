# ase-sdk-go

## Installation
```go
go get github.com/iflytek/ase-sdk-go
```

## Usage

### 非流式
```go
package main

import (
	"fmt"
	"time"

	"github.com/iflytek/ase-sdk-go"
)

func main() {
	cli, err := ase.NewClient(
		"appid",
		"apikey",
		"secret",
		"host",
		"/example",
		ase.WithOnceTimeout(time.Second*3),
		ase.WithOnceRetryCount(3),
		ase.WithTLS(),
	)
	if err != nil {
		panic(err)
	}

	headers := ase.RequestHeader{}
	headers.SetAppID(appid)
	headers.SetStatus(ase.StatusForOnce)
	
	req := new(ase.Request)
	req.SetHeaders(headers)
	req.SetParameters(map[string]interface{}{})
	req.SetPayloads(map[string]interface{}{})

	resp, err := cli.Once(req)
	if err != nil {
		panic(err)
	}

	fmt.Printf("response: %s\n", string(resp))
}
```

### 流式

```go
package main

import (
	"encoding/json"
	"sync"
	"time"

	ase "github.com/iflytek/ase-sdk-go"
)

func main() {
	cli, err := ase.NewClient(
		"appid",
		"apikey",
		"secret",
		"host",
		"/example",
		ase.WithTLS(),
		ase.WithStreamReadTimeout(time.Second*5),
		ase.WithStreamWriteTimeout(time.Second*5),
	)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for {
			msg, err := cli.Receive()
			if err != nil {
				panic(err)
			}

			var resp ase.Resp
			if err = json.Unmarshal(msg, &resp); err != nil {
				panic(err)
			}

			if resp.Header.Status == ase.StatusLastFrame {
				_ = cli.Destroy()
				return
			}	
        }
	}()

	// mock inputs
	for i := 0; i < 100; i++ {
		var status int
		if i == 0 {
			status = ase.StatusFirstFrame
		} else if i == 99 {
			status = ase.StatusLastFrame
		} else {
			status = ase.StatusContinue
		}

		headers := ase.RequestHeader{}
		headers.SetAppID(appid)
		headers.SetStatus(status)
		
		req := new(ase.Request)
		req.SetHeaders(headers)
		req.SetParameters(map[string]interface{}{})
		req.SetPayloads(map[string]interface{}{})

		if err = cli.Send(req); err != nil {
			panic(err)
		}
	}
	
	wg.Wait()
}


```