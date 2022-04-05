package goservice

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
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
	CalledTime    int64       `json:"called_time"`
}

type ResponseTranferData struct {
	Data            interface{} `json:"data"`
	Error           bool        `json:"error"`
	ResponseId      string      `json:"response_id"`
	ResponseNodeId  string      `json:"response_node_id"`
	ResponseService string      `json:"response_service"`
	ResponseAction  string      `json:"response_action"`
	ResponseTime    int64       `json:"response_time"`
}
type Transporter struct {
	Config    TransporterConfig
	Subscribe func(channel string) interface{}
	Emit      func(channel string, data interface{}) error
	Receive   func(func(string, RequestTranferData, error), interface{})
}

var transporter Transporter
var bus EventBus.Bus

func initTransporter() {
	bus = EventBus.New()
	transporter = Transporter{
		Config: broker.Config.TransporterConfig,
	}

	switch transporter.Config.TransporterType {
	case TransporterTypeRedis:
		initRedisTransporter()
		break
	}
}

func initRedisTransporter() {
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

	// subscribe channel
}

func listenActionCall(serviceName string, action Action) {
	switch transporter.Config.TransporterType {
	case TransporterTypeRedis:
		redisListenActionCall(serviceName, action.Name)
		break
	}
	channel := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + serviceName + "." + action.Name
	bus.Subscribe(channel, func(data RequestTranferData) {
		responseId := data.ResponseId
		ctx := Context{
			RequestId:   data.CallerNodeId,
			ResponseId:  uuid.New().String(),
			Params:      data.Params,
			Meta:        data.Meta,
			FromNode:    data.CallerNodeId,
			FromService: data.CallerService,
			FromAction:  data.CallerAction,
			Call: func(action string, params, meta interface{}) (interface{}, error) {
				return nil, nil
			},
		}

		// handle action
		res, e := action.Handle(&ctx)

		// response result
		responseTranferData := ResponseTranferData{
			ResponseId:      responseId,
			ResponseNodeId:  broker.Config.NodeId,
			ResponseService: serviceName,
			ResponseAction:  action.Name,
			ResponseTime:    time.Now().UnixNano(),
		}

		if e != nil {
			responseTranferData.Error = true
			responseTranferData.Data = nil
		} else {
			responseTranferData.Error = false
			responseTranferData.Data = res
		}

		responseChanel := channel + ".response"
		bus.Publish(responseChanel, responseTranferData)
	})
}

func redisListenActionCall(serviceName string, actionName string) {
	channelTransporter := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".request"
	channelInternal := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + serviceName + "." + actionName
	pb := transporter.Subscribe(channelTransporter)
	pubsub := pb.(*redis.PubSub)
	transporter.Receive(func(cn string, data RequestTranferData, err error) {
		bus.Publish(channelInternal, data)
	}, pubsub)
}

// calling
func callAction(ctx Context, actionName string, params interface{}, meta interface{}) (interface{}, error) {
	data := make(chan ResponseTranferData, 1)
	var err error
	go func() {
		// loop
		var action RegistryAction
		var service RegistryService
		for _, s := range registryServices {
			for _, a := range s.Actions {
				if actionName == s.Name+"."+a.Name {
					action = a
					service = s
				}
			}
		}
		if action.Name == "" && service.Name == "" {
			err = errors.New("Action or event `" + action.Name + "` is not existed")
			data = nil
		} else {
			// Init data send
			channelTransporter := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".request"
			channelInternal := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + service.Name + "." + action.Name
			responseId := uuid.New().String()
			dataSend := RequestTranferData{
				Params:        params,
				Meta:          meta,
				RequestId:     ctx.RequestId,
				ResponseId:    responseId,
				CallerNodeId:  broker.Config.NodeId,
				CallerService: service.Name,
				CallerAction:  action.Name,
				CallingLevel:  ctx.CallingLevel + 1,
				CalledTime:    time.Now().UnixNano(),
			}
			// Subscribe response data
			bus.SubscribeOnce(channelInternal, func(d ResponseTranferData) {
				data <- d
			})

			// push transporter
			transporter.Emit(channelTransporter, dataSend)
		}
	}()

	select {
	case res := <-data:
		if err != nil {
			return nil, err
		}
		return res, nil
	case <-time.After(time.Duration(broker.Config.RequestTimeOut) * time.Millisecond):
		return nil, errors.New("Timeout")
	}
}
