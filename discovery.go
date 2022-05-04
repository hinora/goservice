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
	Enable                   bool
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

const (
	channelGlobalDiscovery  = GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcasts)
	channelGlobalInfo       = GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsInfo)
	channelGlobalHeartBeat  = GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsHeartbeat)
	channelGlobalDisconnect = GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsDisconnect)
)

func (b *Broker) initDiscovery() {
	b.channelPrivateInfo = GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsInfo) + "." + b.Config.NodeId

	ip, err := b.getOutboundIP()
	if err != nil {
		panic(err)
	}
	b.registryNode = RegistryNode{
		NodeId: b.Config.NodeId,
		IP:     []string{ip.String()},
	}
	b.registryNodes = []RegistryNode{}
}

func (b *Broker) startDiscovery() {
	// init count metrics
	b.initMestricCountCallAction()
	// emit service info to event internal
	b.emitServiceInfoInternal()
	if !b.Config.DiscoveryConfig.Enable {
		return
	}
	b.LogInfo("Discovery start")

	// init service info
	switch b.Config.DiscoveryConfig.DiscoveryType {
	case DiscoveryTypeRedis:
		config := b.Config.DiscoveryConfig.Config.(DiscoveryRedisConfig)
		rdb := redis.NewClient(&redis.Options{
			Addr:     config.Host + ":" + strconv.Itoa(config.Port),
			Password: config.Password,
			DB:       config.Db,
		})

		// start listen
		go b.listenDiscoveryRedis(rdb)
		go b.listenDiscoveryGlobalRedis(rdb)
		// broadcast info
		b.broadcastGlobal(rdb)

		// clear node timeout
		b.clearNodeTimeout()
		break
	}

}

func (b *Broker) listenDiscoveryGlobalRedis(rdb *redis.Client) {
	var ctx = context.Background()
	// listen discovery
	go func() {
		pubsub := rdb.Subscribe(ctx, channelGlobalDiscovery)
		defer pubsub.Close()
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				panic(err)
			}
			deJ, e := DeSerializerJson(msg.Payload)
			if e == nil {
				var topicDiscoveryData = TopicDiscoveryData{}
				mapstructure.Decode(deJ, &topicDiscoveryData)

				if topicDiscoveryData.Sender.NodeId == b.Config.NodeId {
					continue
				}

				// register node
				checkNode := false
				for _, n := range b.registryNodes {
					if n.NodeId == topicDiscoveryData.Sender.NodeId {
						checkNode = true
					}
				}
				if !checkNode {
					topicDiscoveryData.Sender.LastActive = int(time.Now().UnixMilli())
					b.registryNodes = append(b.registryNodes, topicDiscoveryData.Sender)
				}

				// emit register service
				info := TopicInfoData{}
				info.Sender.IP = b.registryNode.IP
				info.Sender.NodeId = b.Config.NodeId

				for _, s := range b.Services {
					var registryActions []RegistryAction
					for _, a := range s.Actions {
						registryActions = append(registryActions, RegistryAction{
							Name:   a.Name,
							Params: a.Params,
							Rest:   a.Rest,
						})
					}
					var registryEvents []RegistryEvent
					for _, e := range s.Events {
						registryEvents = append(registryEvents, RegistryEvent{
							Name:   e.Name,
							Params: e.Params,
						})
					}
					info.Services = append(info.Services, RegistryService{
						Node:    b.registryNode,
						Name:    s.Name,
						Actions: registryActions,
						Events:  registryEvents,
					})
				}

				// response info
				channel := GO_SERVICE_PREFIX + "." + string(DiscoveryBroadcastsInfo) + "." + topicDiscoveryData.Sender.NodeId
				infoSeri, _ := SerializerJson(info)
				rdb.Publish(ctx, channel, infoSeri)
				b.LogInfo("Node `" + topicDiscoveryData.Sender.NodeId + "` connected")
			}
		}
	}()
	// listen info
	go func() {
		pubsub := rdb.Subscribe(ctx, channelGlobalInfo)
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
				if topicInfoData.Sender.NodeId == b.Config.NodeId {
					continue
				}
				// register
				for _, rgi := range topicInfoData.Services {
					check := false
					for _, rgp := range b.registryServices {
						if rgi.Node.NodeId == rgp.Node.NodeId && rgi.Name == rgp.Name {
							check = true
							break
						}
					}
					if !check {
						b.registryServices = append(b.registryServices, rgi)
					}
				}
				b.initMestricCountCallAction()

				// emit service info to event internal
				b.emitServiceInfoInternal()
				b.LogInfo("Receive info from `" + topicInfoData.Sender.NodeId + "`")
			}
		}
	}()
	// listen disconnect
	go func() {
		pubsub := rdb.Subscribe(ctx, channelGlobalDisconnect)
		defer pubsub.Close()
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				panic(err)
			}
			deJ, e := DeSerializerJson(msg.Payload)

			if e == nil {
				var topicDiscoveryData = TopicDiscoveryData{}
				mapstructure.Decode(deJ, &topicDiscoveryData)
				if topicDiscoveryData.Sender.NodeId == b.Config.NodeId {
					continue
				}

				// remove node
				for i, n := range b.registryNodes {
					if n.NodeId == topicDiscoveryData.Sender.NodeId {
						b.registryNodes = append(b.registryNodes[:i], b.registryNodes[i+1:]...)
						continue
					}
				}

				// remove service
				tempRegistryServices := []RegistryService{}
				for _, rgp := range b.registryServices {
					if rgp.Node.NodeId != topicDiscoveryData.Sender.NodeId || rgp.Node.NodeId == b.Config.NodeId {
						tempRegistryServices = append(tempRegistryServices, rgp)
					}
				}
				b.registryServices = tempRegistryServices
				b.LogInfo("Node `" + topicDiscoveryData.Sender.NodeId + "` disconnected")

				// emit service info to event internal
				b.emitServiceInfoInternal()
			}
		}
	}()
	// listen heartbeat
	go func() {
		pubsub := rdb.Subscribe(ctx, channelGlobalHeartBeat)
		defer pubsub.Close()
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				panic(err)
			}
			deJ, e := DeSerializerJson(msg.Payload)
			if e == nil {
				var topicHeartbeatData = TopicHeartbeatData{}
				mapstructure.Decode(deJ, &topicHeartbeatData)

				if topicHeartbeatData.Sender.NodeId == b.Config.NodeId {
					continue
				}

				// update node
				for i := 0; i < len(b.registryNodes); i++ {
					if b.registryNodes[i].NodeId == topicHeartbeatData.Sender.NodeId {
						b.registryNodes[i].LastActive = int(time.Now().UnixMilli())
					}
				}
			}
		}
	}()
}
func (b *Broker) listenDiscoveryRedis(rdb *redis.Client) {
	var ctx = context.Background()

	// listen info
	go func() {
		pubsub := rdb.Subscribe(ctx, b.channelPrivateInfo)
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

				// register node
				checkNode := false
				for _, n := range b.registryNodes {
					if n.NodeId == topicInfoData.Sender.NodeId {
						checkNode = true
					}
				}
				if !checkNode {
					topicInfoData.Sender.LastActive = int(time.Now().UnixMilli())
					b.registryNodes = append(b.registryNodes, topicInfoData.Sender)
				}
				// register
				for _, rgi := range topicInfoData.Services {
					check := false
					for _, rgp := range b.registryServices {
						if rgi.Node.NodeId == rgp.Node.NodeId && rgi.Name == rgp.Name {
							check = true
							break
						}
					}
					if !check {
						b.registryServices = append(b.registryServices, rgi)
					}
				}
				b.initMestricCountCallAction()
				b.LogInfo("Receive info from `" + topicInfoData.Sender.NodeId + "`")

				// emit service info to event internal
				b.emitServiceInfoInternal()
			}
		}
	}()
}

func (b *Broker) broadcastGlobal(rdb *redis.Client) {
	// publish discovery
	go func() {
		var ctx = context.Background()
		info, _ := SerializerJson(TopicDiscoveryData{
			Sender: b.registryNode,
		})
		err := rdb.Publish(ctx, channelGlobalDiscovery, info).Err()
		if err != nil {
			panic(err)
		}
	}()
	// publish info
	go func() {
		var ctx = context.Background()
		info, _ := SerializerJson(TopicInfoData{
			Sender:   b.registryNode,
			Services: b.registryServices,
		})
		err := rdb.Publish(ctx, channelGlobalInfo, info).Err()
		if err != nil {
			panic(err)
		}
	}()
	// heartbeat
	go func() {
		for {
			time.Sleep(time.Millisecond * time.Duration(b.Config.DiscoveryConfig.HeartbeatInterval))

			var ctx = context.Background()
			info, _ := SerializerJson(TopicHeartbeatData{
				Sender: b.registryNode,
			})
			err := rdb.Publish(ctx, channelGlobalHeartBeat, info).Err()
			if err != nil {
				panic(err)
			}
		}
	}()
}

func (b *Broker) clearNodeTimeout() {
	go func() {
		for {
			time.Sleep(time.Second * 2)
			now := time.Now().UnixMilli()
			var tempNodes []RegistryNode
			checkNodeTimeOut := false
			for _, n := range b.registryNodes {
				if now-int64(n.LastActive) <= int64(b.Config.DiscoveryConfig.CleanOfflineNodesTimeout) {
					tempNodes = append(tempNodes, n)
				} else {
					checkNodeTimeOut = true
					b.LogInfo("Node `" + n.NodeId + "` timeout. Removed")
				}
			}
			if checkNodeTimeOut {
				b.registryNodes = tempNodes
				var tempServices []RegistryService
				for _, s := range b.registryServices {
					check := false
					for _, n := range b.registryNodes {
						if n.NodeId == s.Node.NodeId {
							check = true
						}
					}
					if check || s.Node.NodeId == b.Config.NodeId {
						tempServices = append(tempServices, s)
					}
				}
				b.registryServices = tempServices
				b.emitServiceInfoInternal()
			}
		}
	}()
}

// Get preferred outbound ip of this machine
func (b *Broker) getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
