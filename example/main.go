package main

import (
	"fmt"
	"time"

	"github.com/hinora/goservice"
)

func main() {
	// ser, _ := goservice.SerializerJson(goservice.RequestTranferData{
	// 	Params:        nil,
	// 	Meta:          nil,
	// 	RequestId:     "asd",
	// 	ResponseId:    "dsa",
	// 	CallerNodeId:  "Dfds",
	// 	CallerService: "service",
	// 	CallerAction:  "action",
	// 	CallingLevel:  1,
	// 	CalledTime:    time.Now().UnixNano(),
	// })
	// de, _ := goservice.DeSerializerJson(ser)
	// fmt.Println(ser)
	// fmt.Println(de)
	goservice.Init()
	goservice.LoadService(goservice.Service{
		Name: "math",
		Actions: []goservice.Action{
			{
				Name:   "plus",
				Params: map[string]interface{}{},
				Handle: func(context *goservice.Context) (interface{}, error) {
					fmt.Println("Handle action plus")
					return "test", nil
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
