package goservice

import (
	"context"
	"strconv"

	"github.com/go-redis/redis/v8"
)

type PackageType int

const (
	PackageRequest PackageType = iota + 1
	PackageResponse
	PackageEvent
)

type TransporterType int

const (
	TransporterTypeRedis TransporterType = iota + 1
)

type TransporterRedisConfig struct {
	Port     int
	Host     string
	Password string
	Db       int
}

type TransporterConfig struct {
	TransporterType TransporterType
	Config          interface{}
}

type Transporter struct {
	Config    TransporterConfig
	Subscribe func(channel string) *redis.PubSub
	Emit      func(channel string, data interface{}) error
	Receive   func(*redis.PubSub, func(string, interface{}, error))
}

var transporter Transporter

func initTransporter() {
	transporter = Transporter{
		Config: broker.Config.TransporterConfig,
	}

	switch transporter.Config.TransporterType {
	case TransporterTypeRedis:
		// redis transporter
		var ctx = context.Background()
		config := broker.Config.TransporterConfig.Config.(TransporterRedisConfig)
		rdb := redis.NewClient(&redis.Options{
			Addr:     config.Host + ":" + strconv.Itoa(config.Port),
			Password: config.Password,
			DB:       config.Db,
		})
		transporter.Subscribe = func(channel string) *redis.PubSub {
			pubsub := rdb.Subscribe(ctx, channel)
			return pubsub
		}
		transporter.Emit = func(channel string, data interface{}) error {
			data, err := SerializerJson(data)
			if err != nil {
				return err
			}
			err = rdb.Publish(ctx, channel, data).Err()
			if err != nil {
				return err
			}

			return nil
		}
		transporter.Receive = func(pubsub *redis.PubSub, callBack func(string, interface{}, error)) {
			for {
				msg, err := pubsub.ReceiveMessage(ctx)
				if err != nil {
					panic(err)
				}

				data, err := DeSerializerJson(msg.Payload)
				if err != nil {
					callBack("", nil, err)
				}
				callBack(msg.Channel, data, nil)
			}
		}
		break
	}
}

func listenActionCall(serviceName string, action *Action) {
	// var ctx = context.Background()
	// channel := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + serviceName + "."
	// pubsub := rdb.Subscribe(ctx, channel)
	// defer pubsub.Close()
	// for {
	// 	msg, err := pubsub.ReceiveMessage(ctx)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	fmt.Println(msg.Channel, msg.Payload)
	// }
}
