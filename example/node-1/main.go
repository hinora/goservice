package main

import (
	"fmt"
	"time"

	"github.com/hinora/goservice"
)

func main() {
	goservice.Init(goservice.BrokerConfig{
		NodeId: "Node-1",
		DiscoveryConfig: goservice.DiscoveryConfig{
			DiscoveryType: goservice.DiscoveryTypeRedis,
			Config: goservice.DiscoveryRedisConfig{
				Port: 6379,
				Host: "127.0.0.1",
			},
			HeartbeatInterval:        3000,
			HeartbeatTimeout:         7000,
			CleanOfflineNodesTimeout: 9000,
		},
		TransporterConfig: goservice.TransporterConfig{
			TransporterType: goservice.TransporterTypeRedis,
			Config: goservice.TransporterRedisConfig{
				Port: 6379,
				Host: "127.0.0.1",
			},
		},
		RequestTimeOut: 5000,
		TraceConfig: goservice.TraceConfig{
			Enabled:      true,
			TraceExpoter: goservice.TraceExporterConsole,
		},
	})
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
			// data2, err2 := ctx.Call("hello.say_hi", nil, nil)
			// fmt.Println("Response from say hi: ", data2, err2)
		},
	})
	goservice.Hold()
}
