package main

import (
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

	b.LoadService(&goservice.Service{
		Name: "hello",
		Actions: []goservice.Action{
			{
				Name:   "say_hi",
				Params: map[string]interface{}{},
				Rest: goservice.Rest{
					Method: goservice.GET,
					Path:   "/say_hi",
				},
				Handle: func(ctx *goservice.Context) (interface{}, error) {
					ctx.LogInfo("Handle action say hi from node 2")
					var wg sync.WaitGroup
					totalCall := 2
					for i := 0; i < totalCall; i++ {
						wg.Add(1)
					}
					for j := 0; j < totalCall; j++ {
						go func() {
							defer wg.Done()
						}()
					}
					wg.Wait()
					ctx.Call("event.test", nil, nil)
					return "This is result from action say hi", nil
				},
			},
		},
		Events: []goservice.Event{
			{
				Name: "event.test",
				Handle: func(ctx *goservice.Context) {
					ctx.LogInfo("Handle event test from node 2")
				},
			},
		},
	})
	b.Hold()
}
