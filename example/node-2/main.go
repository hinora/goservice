package main

import (
	"time"

	"github.com/hinora/goservice"
)

func main() {
	b := goservice.Init(goservice.BrokerConfig{
		NodeId: "Node-2",
		DiscoveryConfig: goservice.DiscoveryConfig{
			Enable:        true,
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
			Enable:          true,
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
		LoggerConfig: goservice.Logconfig{
			Enable:   true,
			Type:     goservice.LogConsole,
			LogLevel: goservice.LogTypeInfo,
		},
	})
	b.LoadService(&goservice.Service{
		Name: "test",
		Actions: []goservice.Action{
			{
				Name:   "test",
				Params: map[string]interface{}{},
				Handle: func(ctx *goservice.Context) (interface{}, error) {
					// ctx.LogWarning("Handle action plus")
					time.Sleep(time.Second * 1)
					ctx.Meta = map[string]interface{}{
						"test": "aaa",
					}
					ctx.Call("test.test2", nil)
					return ctx.Params, nil
				},
			},
			{
				Name:   "test2",
				Params: map[string]interface{}{},
				Handle: func(ctx *goservice.Context) (interface{}, error) {
					// ctx.LogWarning("Handle action minus")
					// fmt.Println("meta incoming: ", ctx.Meta)
					time.Sleep(time.Second * 1)
					return "asdasd", nil
				},
			},
		},
		Events: []goservice.Event{
			{
				Name: "event.test",
				Handle: func(ctx *goservice.Context) {
					ctx.LogInfo("Handle event test from node 1")
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

	b.Hold()
}
