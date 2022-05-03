package main

import (
	"fmt"
	"time"

	"github.com/hinora/goservice"
)

func main() {
	b := goservice.Init(goservice.BrokerConfig{
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
	b.LoadService(&goservice.Service{
		Name: "math",
		Actions: []goservice.Action{
			{
				Name:   "plus",
				Params: map[string]interface{}{},
				Rest: goservice.Rest{
					Method: goservice.GET,
					Path:   "/plus",
				},
				Handle: func(context *goservice.Context) (interface{}, error) {
					fmt.Println("Handle action plus")
					time.Sleep(time.Second * 1)
					return "This is result from action math.plus", nil
				},
			},
		},
		Events: []goservice.Event{
			{
				Name: "event.test",
				Handle: func(context *goservice.Context) {
					fmt.Println("Handle event test from node 1")
				},
			},
		},
		// Started: func(ctx *goservice.Context) {
		// 	time.Sleep(time.Millisecond * 5000)
		// 	fmt.Println("service test started")

		// 	fmt.Println(ctx.Call("event.test", nil, nil))
		// 	data, err := ctx.Call("hello.say_hi", nil, nil)
		// 	fmt.Println("Response from say hi: ", data, err)
		// 	// data2, err2 := ctx.Call("hello.say_hi", nil, nil)
		// 	// fmt.Println("Response from say hi: ", data2, err2)
		// },
	})

	// gateway
	gateway := goservice.InitGateway(goservice.GatewayConfig{
		Name: "api",
		Host: "127.0.0.1",
		Port: 8000,
		Routes: []goservice.GatewayConfigRoute{
			{
				Path: "/api",
				WhileList: []string{
					".*",
				},
			},
		},
	})
	b.LoadService(gateway)
	b.Hold()
}
