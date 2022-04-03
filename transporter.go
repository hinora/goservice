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
type RequestTranferData struct {
	Params        interface{} `json:"params"`
	Meta          interface{} `json:"meta"`
	RequestId     string      `json:"request_id"`
	ResponseId    string      `json:"response_id"`
	TraceParentId string      `json:"trace_parent_id"`
	CallerNodeId  string      `json:"caller_node_id"`
	CallerService string      `json:"caller_service"`
	CallerAction  string      `json:"caller_action"`
	CallingLevel  int         `json:"calling_level"`
}

type Transporter struct {
	Config    TransporterConfig
	Subscribe func(channel string) interface{}
	Emit      func(channel string, data interface{}) error
	Receive   func(func(string, RequestTranferData, error), interface{})
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
		transporter.Subscribe = func(channel string) interface{} {
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
		transporter.Receive = func(callBack func(string, RequestTranferData, error), pubsub interface{}) {
			ps := pubsub.(*redis.PubSub)
			for {
				msg, err := ps.ReceiveMessage(ctx)
				if err != nil {
					panic(err)
				}

				data, err := DeSerializerJson(msg.Payload)
				if err != nil {
					callBack("", RequestTranferData{}, err)
				}
				callBack(msg.Channel, data.(RequestTranferData), nil)
			}
		}
		break
	}
}

func listenActionCall(serviceName string, action Action) {
	channel := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + serviceName + "." + action.Name
	switch transporter.Config.TransporterType {
	case TransporterTypeRedis:
		pb := transporter.Subscribe(channel)
		pubsub := pb.(*redis.PubSub)
		transporter.Receive(func(cn string, data RequestTranferData, err error) {
			logInfo("Call to channel: " + cn)
			responseId := data.ResponseId
			ctx := Context{
				RequestId:   data.CallerNodeId,
				ResponseId:  data.ResponseId,
				Params:      data.Params,
				Meta:        data.Meta,
				FromNode:    data.CallerNodeId,
				FromService: data.CallerService,
				FromAction:  data.CallerAction,
				Call: func(action string, params, meta interface{}) (interface{}, error) {
					return nil, nil
				},
			}
			res, err := action.Handle(&ctx)
		}, pubsub)
		break
	}
}
