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
					wg.Add(1)
					wg.Add(1)
					go func() {
						defer wg.Done()
						data, err := ctx.Call("math.plus", nil, nil)
						fmt.Println("Response from math.plus: ", data, err)
					}()
					go func() {
						defer wg.Done()
						data1, err1 := ctx.Call("math.plus", nil, nil)
						fmt.Println("Response from math.plus: ", data1, err1)
					}()
					wg.Wait()
					return "This is result from action say hi", nil
				},
			},
		},
	})
	b.Hold()
}
