package goservice

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
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
	Config                   interface{}
	DiscoveryType            DiscoveryType
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
	Sender RegistryNode `json:"sender" mapstructure:"sender"`
}

type TopicInfoData struct {
	Sender   RegistryNode      `json:"sender" mapstructure:"sender"`
	Services []RegistryService `json:"services" mapstructure:"services"`
}

type TopicHeartbeatData struct {
	Sender RegistryNode `json:"sender" mapstructure:"sender"`
	Cpu    float64      `json:"cpu" mapstructure:"cpu"`
	Ram    int          `json:"ram" mapstructure:"ram"`
}

type TopicPingData struct {
	Sender RegistryNode `json:"sender" mapstructure:"sender"`
	Time   uint64       `json:"time" mapstructure:"time"`
}

type TopicPongData struct {
	Sender  RegistryNode `json:"sender" mapstructure:"sender"`
	Time    uint64       `json:"time" mapstructure:"time"`
	Arrived uint64       `json:"arrived" mapstructure:"arrived"`
}

type TopicDisconnectData struct {
	Sender  RegistryNode `json:"sender" mapstructure:"sender"`
	Time    uint64       `json:"time" mapstructure:"time"`
	Arrived uint64       `json:"arrived" mapstructure:"arrived"`
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

var registryServices []RegistryService
var registryNode []RegistryNode

func initDiscovery() {
	ip, err := getOutboundIP()
	if err != nil {
		panic(err)
	}
	registryNode = []RegistryNode{
		{
			NodeId: broker.Config.NodeId,
			IP:     []string{ip.String()},
		},
	}
}

func startDiscovery() {
	switch broker.Config.DiscoveryConfig.DiscoveryType {
	case DiscoveryTypeRedis:
		config := broker.Config.DiscoveryConfig.Config.(DiscoveryRedisConfig)
		rdb := redis.NewClient(&redis.Options{
			Addr:     config.Host + ":" + strconv.Itoa(config.Port),
			Password: config.Password,
			DB:       config.Db,
		})

		// start listen
		go listenDiscoveryRedis(rdb)
		time.Sleep(time.Millisecond * 1000)
		// broadcast info
		broadcastRegistryInfo(rdb)
		break
	}

}

func listenDiscoveryRedis(rdb *redis.Client) {
	logInfo("Discovery listener started")
	var ctx = context.Background()
	channel := GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsInfo)
	pubsub := rdb.Subscribe(ctx, channel)
	defer pubsub.Close()
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}
		deJ, e := DeSerializerJson(msg.Payload)
		if e == nil {
			var topicInfoData = TopicInfoData{}
			mapstructure.Decode(deJ, &topicInfoData)
			services := topicInfoData.Services

			for _, service := range services {
				var registryActions []RegistryAction
				for _, a := range service.Actions {
					registryActions = append(registryActions, RegistryAction{
						Name:   a.Name,
						Params: a.Params,
					})
				}
				var registryEvents []RegistryEvent
				for _, e := range service.Events {
					registryEvents = append(registryEvents, RegistryEvent{
						Name:   e.Name,
						Params: e.Params,
					})
				}

				registryServices = append(registryServices, RegistryService{
					Node:    service.Node,
					Name:    service.Name,
					Actions: registryActions,
					Events:  registryEvents,
				})
			}
		}
	}
}

func broadcastRegistryInfo(rdb *redis.Client) {
	logInfo("Discovery emit info service")
	var ctx = context.Background()
	channel := GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsInfo)

	info, _ := SerializerJson(TopicInfoData{
		Sender: RegistryNode{
			NodeId: broker.Config.NodeId,
		},
		Services: registryServices,
	})
	err := rdb.Publish(ctx, channel, info).Err()
	if err != nil {
		panic(err)
	}
}

// Get preferred outbound ip of this machine
func getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
