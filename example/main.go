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
					return "This is result from action math.plus", nil
				},
			},
		},
		Started: func(ctx *goservice.Context) {
			time.Sleep(time.Millisecond * 5000)
			fmt.Println("service test started")

			data, err := ctx.Call("hello.say_hi", nil, nil)
			fmt.Println("Response from say hi: ", data, err)
		},
	})
	goservice.LoadService(goservice.Service{
		Name: "hello",
		Actions: []goservice.Action{
			{
				Name:   "say_hi",
				Params: map[string]interface{}{},
				Handle: func(ctx *goservice.Context) (interface{}, error) {
					fmt.Println("Handle action say hi")
					data, err := ctx.Call("math.plus", nil, nil)
					fmt.Println("Response from math.plus: ", data, err)
					return "This is result from action say hi", nil
				},
			},
		},
	})
	goservice.Hold()
}
