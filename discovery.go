package goservice

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// DISCOVERY & REGISTRY
type DiscoveryType int

const (
	DiscoveryTypeRedis DiscoveryType = iota + 1
)

type DiscoveryRedisConfig struct {
	Port     int
	Host     string
	Password string
	Db       int
}
type DiscoveryConfig struct {
	HeartbeatInterval        int
	HeartbeatTimeout         int
	CleanOfflineNodesTimeout int
}

type DiscoveryPotocol int

const (
	TopicDiscover DiscoveryPotocol = iota + 1
	TopicInfo
	TopicHeartbeat
	TopicPing
	TopicPong
	TopicDisconnect
)

type TopicDiscoveryData struct {
	Sender RegistryNode
}

type TopicInfoData struct {
	Sender   RegistryNode
	Services []RegistryService
}

type TopicHeartbeatData struct {
	Sender RegistryNode
	Cpu    float64
	Ram    int
}

type TopicPingData struct {
	Sender RegistryNode
	Time   uint64
}

type TopicPongData struct {
	Sender  RegistryNode
	Time    uint64
	Arrived uint64
}

type TopicDisconnectData struct {
	Sender  RegistryNode
	Time    uint64
	Arrived uint64
}

type DiscoveryBroadcastsChannelType string

const (
	DiscoveryBroadcasts           DiscoveryBroadcastsChannelType = "DISCOVERY"
	DiscoveryBroadcastsInfo       DiscoveryBroadcastsChannelType = "INFO"
	DiscoveryBroadcastsHeartbeat  DiscoveryBroadcastsChannelType = "HEART_BEAT"
	DiscoveryBroadcastsPing       DiscoveryBroadcastsChannelType = "PING"
	DiscoveryBroadcastsPong       DiscoveryBroadcastsChannelType = "PONG"
	DiscoveryBroadcastsDisconnect DiscoveryBroadcastsChannelType = "DISCONNECT"
)

var registryService RegistryService
var registryNode RegistryNode

func StartDiscovery(typeDiscovery DiscoveryType, config interface{}) {
	switch typeDiscovery {
	case DiscoveryTypeRedis:
		config := config.(DiscoveryRedisConfig)
		rdb := redis.NewClient(&redis.Options{
			Addr:     config.Host + ":" + strconv.Itoa(config.Port),
			Password: config.Password,
			DB:       config.Db,
		})

		// start listen
		go listenDiscoveryRedis(rdb)
		// broadcast info
		broadcastRegistryInfo(rdb)
		break
	}

}

func listenDiscoveryRedis(rdb *redis.Client) {
	var ctx = context.Background()
	channel := GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsInfo)
	pubsub := rdb.Subscribe(ctx, channel)
	defer pubsub.Close()
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}

		fmt.Println(msg.Channel, msg.Payload)
	}
}

func broadcastRegistryInfo(rdb *redis.Client) {
	var ctx = context.Background()
	channel := GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsInfo)

	info, _ := SerializerJson(TopicInfoData{
		Sender: RegistryNode{
			NodeId: "Test",
		},
		Services: []RegistryService{
			{
				Node: RegistryNode{
					NodeId: "Test",
				},
				Name: "service_1",
			},
		},
	})
	err := rdb.Publish(ctx, channel, info).Err()
	if err != nil {
		panic(err)
	}
}
