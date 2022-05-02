package main

import (
	"fmt"
	"sync"

	"github.com/hinora/goservice"
)

func main() {
	b := goservice.Init(goservice.BrokerConfig{
		NodeId: "Node-2",
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

	b.LoadService(goservice.Service{
		Name: "hello",
		Actions: []goservice.Action{
			{
				Name:   "say_hi",
				Params: map[string]interface{}{},
				Handle: func(ctx *goservice.Context) (interface{}, error) {
					fmt.Println("Handle action say hi from node 2")
					var wg sync.WaitGroup
					totalCall := 100
					for i := 0; i < totalCall; i++ {
						wg.Add(1)
					}
					for j := 0; j < totalCall; j++ {
						callTime := j
						go func() {
							defer wg.Done()
							data, err := ctx.Call("math.plus", nil, nil)
							fmt.Print("Call time: ", callTime)
							fmt.Print(" - Response from math.plus: ", data, err)
							fmt.Println()
						}()
					}
					wg.Wait()
					// ctx.Call("event.test", nil, nil)
					return "This is result from action say hi", nil
				},
			},
		},
		Events: []goservice.Event{
			{
				Name: "event.test",
				Handle: func(context *goservice.Context) {
					fmt.Println("Handle event test from node 2")
				},
			},
			{
				Name: "service.info",
				Handle: func(context *goservice.Context) {
					fmt.Println("Info: ", context.Params)
				},
			},
		},
	})
	b.Hold()
}
