package main

import (
	"github.com/hinora/goservice"
)

func main() {

	goservice.StartDiscovery(goservice.DiscoveryTypeRedis, goservice.DiscoveryRedisConfig{
		Port: 6379,
		Host: "127.0.0.1",
	})
	goservice.Hold()
}
