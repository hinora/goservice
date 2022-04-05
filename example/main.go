package main

import (
	"fmt"
	"time"

	"github.com/hinora/goservice"
)

func main() {
	goservice.Init()
	goservice.LoadService(goservice.Service{
		Name: "math",
		Actions: []goservice.Action{
			{
				Name:   "plus",
				Params: map[string]interface{}{},
				Handle: func(context *goservice.Context) (interface{}, error) {
					fmt.Println("Handle action plus")
					return nil, nil
				},
			},
		},
		Started: func(ctx *goservice.Context) {
			time.Sleep(time.Millisecond * 5000)
			fmt.Println("service test started")
			data, err := ctx.Call("math.plus", nil, nil)
			fmt.Println("Response: ", data, err)
		},
	})
	goservice.Hold()
}
