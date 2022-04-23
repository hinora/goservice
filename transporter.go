package goservice

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
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
	Params            interface{} `json:"params" mapstructure:"params"`
	Meta              interface{} `json:"meta" mapstructure:"meta"`
	RequestId         string      `json:"request_id" mapstructure:"request_id"`
	ResponseId        string      `json:"response_id" mapstructure:"response_id"`
	TraceParentId     string      `json:"trace_parent_id" mapstructure:"trace_parent_id"`
	TraceRootParentId string      `json:"trace_root_parent_id" mapstructure:"trace_root_parent_id"`
	CallerNodeId      string      `json:"caller_node_id" mapstructure:"caller_node_id"`
	CallerService     string      `json:"caller_service" mapstructure:"caller_service"`
	CallerAction      string      `json:"caller_action" mapstructure:"caller_action"`
	CallingLevel      int         `json:"calling_level" mapstructure:"calling_level"`
	CalledTime        int64       `json:"called_time" mapstructure:"called_time"`
	CallToService     string      `json:"call_to_service" mapstructure:"call_to_service"`
	CallToAction      string      `json:"call_to_action" mapstructure:"call_to_action"`
}

type ResponseTranferData struct {
	Data            interface{} `json:"data" mapstructure:"data"`
	Error           bool        `json:"error" mapstructure:"error"`
	ErrorMessage    string      `json:"error_message" mapstructure:"error_message"`
	ResponseId      string      `json:"response_id" mapstructure:"response_id"`
	ResponseNodeId  string      `json:"response_node_id" mapstructure:"response_node_id"`
	ResponseService string      `json:"response_service" mapstructure:"response_service"`
	ResponseAction  string      `json:"response_action" mapstructure:"response_action"`
	ResponseTime    int64       `json:"response_time" mapstructure:"response_time"`
	TraceSpans      []traceSpan `json:"trace_spans" mapstructure:"trace_spans"`
}
type Transporter struct {
	Config    TransporterConfig
	Subscribe func(channel string) interface{}
	Emit      func(channel string, data interface{}) error
	Receive   func(func(string, interface{}, error), interface{})
}

var transporter Transporter
var bus EventBus

func initTransporter() {
	bus = EventBus{}
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
	transporter.Receive = func(callBack func(string, interface{}, error), pubsub interface{}) {
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

			callBack(msg.Channel, data, nil)
		}
	}

	// subscribe channel request
	channelRequestTransporter := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".request"
	pbRq := transporter.Subscribe(channelRequestTransporter)
	pubsubRq := pbRq.(*redis.PubSub)
	go transporter.Receive(func(cn string, data interface{}, err error) {
		go func() {
			if err != nil {
				return
			}
			dT := RequestTranferData{}
			mapstructure.Decode(data, &dT)
			responseId := dT.ResponseId

			dT.ResponseId = uuid.New().String()
			// Subscribe response data
			channelCall := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + dT.CallToService + "." + dT.CallToAction
			channelReceive := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".response." + dT.ResponseId
			res, e := emitWithTimeout(channelCall, channelReceive, dT)
			if e != nil {
				return
			}

			channelResponseTransporter := GO_SERVICE_PREFIX + "." + dT.CallerNodeId + ".response"
			resT := ResponseTranferData{}
			mapstructure.Decode(res, &resT)

			resT.ResponseId = responseId
			transporter.Emit(channelResponseTransporter, resT)
		}()
	}, pubsubRq)

	// subceibe channel response
	channelResponseTransporter := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".response"
	pbRs := transporter.Subscribe(channelResponseTransporter)
	pubsubRs := pbRs.(*redis.PubSub)
	go transporter.Receive(func(cn string, data interface{}, err error) {
		go func() {
			if err != nil {
				return
			}
			dRs := ResponseTranferData{}
			mapstructure.Decode(data, &dRs)
			channelResponseTransporter := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".response." + dRs.ResponseId
			bus.Publish(channelResponseTransporter, dRs)
		}()
	}, pubsubRs)

}

func listenActionCall(serviceName string, action Action) {
	channel := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + serviceName + "." + action.Name
	bus.Subscribe(channel, func(data RequestTranferData) {
		go func() {

			responseId := data.ResponseId
			ctx := Context{
				RequestId:         uuid.New().String(),
				ResponseId:        uuid.New().String(),
				Params:            data.Params,
				Meta:              data.Meta,
				FromNode:          data.CallerNodeId,
				FromService:       data.CallerService,
				FromAction:        data.CallerAction,
				CallingLevel:      data.CallingLevel,
				TraceParentId:     data.TraceParentId,
				TraceParentRootId: data.TraceRootParentId,
			}

			// start trace
			spanId := startTraceSpan("Action `"+serviceName+"."+action.Name+"`", "action", serviceName, action.Name, data.Params, ctx.FromNode, ctx.TraceParentId, ctx.CallingLevel, data.TraceRootParentId)

			ctx.TraceParentId = spanId
			ctx.Call = func(a string, params interface{}, meta interface{}) (interface{}, error) {
				callResult, err := callAction(ctx, a, params, meta, serviceName, action.Name)
				if err != nil {
					return nil, err
				}
				return callResult.Data, err
			}

			// handle action
			res, e := action.Handle(&ctx)

			// end trace
			endTraceSpan(spanId, e)

			// response result
			responseTranferData := ResponseTranferData{
				ResponseId:      responseId,
				ResponseNodeId:  broker.Config.NodeId,
				ResponseService: serviceName,
				ResponseAction:  action.Name,
				ResponseTime:    time.Now().UnixNano(),
			}

			// response trace if the exporter is console
			if broker.Config.TraceConfig.TraceExpoter == TraceExporterConsole {
				traceSpans := findTraceChildrensDeep(spanId)
				for _, s := range traceSpans {
					if s.Tags.CallerNodeId != data.CallerNodeId {
						responseTranferData.TraceSpans = append(responseTranferData.TraceSpans, *s)
					}
				}
				trans, errFind := findSpan(spanId)
				if errFind == nil {
					responseTranferData.TraceSpans = append(responseTranferData.TraceSpans, *trans)
				}
			}

			if e != nil {
				responseTranferData.Error = true
				responseTranferData.ErrorMessage = e.Error()
				responseTranferData.Data = nil
			} else {
				responseTranferData.Error = false
				responseTranferData.Data = res
			}

			responseChanel := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".response." + responseId
			bus.Publish(responseChanel, responseTranferData)
		}()
	})
}

// calling
func callAction(ctx Context, actionName string, params interface{}, meta interface{}, callerService string, callerAction string) (ResponseTranferData, error) {
	data := make(chan ResponseTranferData, 1)
	var err error
	channelInternal := ""

	// chose node call
	service, action := balancingRoundRobin(actionName)
	if service.Name == "" && action.Name == "" {
		return ResponseTranferData{}, errors.New("Action or event `" + actionName + "` is not existed")
	}

	go func() {
		// Init data send
		channelTransporter := GO_SERVICE_PREFIX + "." + service.Node.NodeId + ".request"
		responseId := uuid.New().String()
		channelInternal = GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".response." + responseId
		dataSend := RequestTranferData{
			Params:            params,
			Meta:              meta,
			RequestId:         ctx.RequestId,
			ResponseId:        responseId,
			CallerNodeId:      broker.Config.NodeId,
			CallerService:     callerService,
			CallerAction:      callerAction,
			CallingLevel:      ctx.CallingLevel + 1,
			CalledTime:        time.Now().UnixNano(),
			CallToService:     service.Name,
			CallToAction:      action.Name,
			TraceParentId:     ctx.TraceParentId,
			TraceRootParentId: ctx.TraceParentRootId,
		}
		// Subscribe response data
		bus.Subscribe(channelInternal, func(d interface{}) {
			go func() {
				dT := ResponseTranferData{}
				mapstructure.Decode(d, &dT)
				data <- dT
				bus.UnSubscribe(channelInternal)
			}()
		})

		// push transporter
		transporter.Emit(channelTransporter, dataSend)

		// Service in local? Use internal event bus
		// channelCall := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + "." + dT.CallToService + "." + dT.CallToAction
		// channelReceive := GO_SERVICE_PREFIX + "." + broker.Config.NodeId + ".response." + dT.ResponseId
		// res, e := emitWithTimeout(channelCall, channelReceive, dT)
	}()

	select {
	case res := <-data:
		if err != nil {
			return ResponseTranferData{}, err
		}
		addTraceSpans(res.TraceSpans)
		return res, nil
	case <-time.After(time.Duration(broker.Config.RequestTimeOut) * time.Millisecond):
		if channelInternal != "" {
			bus.UnSubscribe(channelInternal)
		}
		err := errors.New("Timeout")
		return ResponseTranferData{}, err
	}
}

func emitWithTimeout(channelCall string, channelReceive string, dataSend interface{}) (interface{}, error) {
	data := make(chan interface{}, 1)
	go func() {

		// Subscribe response data
		bus.Subscribe(channelReceive, func(d interface{}) {
			data <- d
			bus.UnSubscribe(channelReceive)
		})

		bus.Publish(channelCall, dataSend)
	}()
	select {
	case res := <-data:
		return res, nil
	case <-time.After(time.Duration(broker.Config.RequestTimeOut) * time.Millisecond):
		bus.UnSubscribe(channelReceive)
		return nil, errors.New("Timeout")
	}
}
