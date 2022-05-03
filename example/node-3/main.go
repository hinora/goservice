package main

import (
	"fmt"

	"github.com/hinora/goservice"
)

func main() {
	b := goservice.Init(goservice.BrokerConfig{
		NodeId: "Node-3",
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
				Handle: func(ctx *goservice.Context) (interface{}, error) {
					fmt.Println("Handle action say hi from node 3")
					return "This is result from action say hi", nil
				},
			},
		},
	})
	b.Hold()
}
