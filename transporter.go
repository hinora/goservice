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
	CallerEvent       string      `json:"caller_event" mapstructure:"caller_event"`
	CallingLevel      int         `json:"calling_level" mapstructure:"calling_level"`
	CalledTime        int64       `json:"called_time" mapstructure:"called_time"`
	CallToService     string      `json:"call_to_service" mapstructure:"call_to_service"`
	CallToAction      string      `json:"call_to_action" mapstructure:"call_to_action"`
	CallToEvent       string      `json:"call_to_event" mapstructure:"call_to_event"`
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

func (b *Broker) initTransporter() {
	b.bus = initEventBus()
	b.transporter = Transporter{
		Config: b.Config.TransporterConfig,
	}

	switch b.transporter.Config.TransporterType {
	case TransporterTypeRedis:
		b.initRedisTransporter()
		break
	}
}

func (b *Broker) initRedisTransporter() {
	// redis transporter
	var ctx = context.Background()
	config := b.Config.TransporterConfig.Config.(TransporterRedisConfig)
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + strconv.Itoa(config.Port),
		Password: config.Password,
		DB:       config.Db,
	})
	b.transporter.Subscribe = func(channel string) interface{} {
		pubsub := rdb.Subscribe(ctx, channel)
		return pubsub
	}
	b.transporter.Emit = func(channel string, data interface{}) error {
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
	b.transporter.Receive = func(callBack func(string, interface{}, error), pubsub interface{}) {
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
	channelRequestTransporter := GO_SERVICE_PREFIX + "." + b.Config.NodeId + ".request"
	pbRq := b.transporter.Subscribe(channelRequestTransporter)
	pubsubRq := pbRq.(*redis.PubSub)
	go b.transporter.Receive(func(cn string, data interface{}, err error) {
		go func() {
			if err != nil {
				return
			}
			dT := RequestTranferData{}
			mapstructure.Decode(data, &dT)
			responseId := dT.ResponseId
			if dT.CallToAction != "" {

				dT.ResponseId = uuid.New().String()
				// Subscribe response data
				channelCall := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + dT.CallToService + "." + dT.CallToAction
				channelReceive := GO_SERVICE_PREFIX + "." + b.Config.NodeId + ".response." + dT.ResponseId
				res, e := b.emitWithTimeout(channelCall, channelReceive, dT)
				if e != nil {
					return
				}

				channelResponseTransporter := GO_SERVICE_PREFIX + "." + dT.CallerNodeId + ".response"
				resT := ResponseTranferData{}
				mapstructure.Decode(res, &resT)

				resT.ResponseId = responseId
				b.transporter.Emit(channelResponseTransporter, resT)
			} else if dT.CallToEvent != "" {

				channelCall := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + dT.CallToService + "." + dT.CallToEvent
				b.emitWithTimeout(channelCall, "", dT)
			}
		}()
	}, pubsubRq)

	// subscribe channel response
	channelResponseTransporter := GO_SERVICE_PREFIX + "." + b.Config.NodeId + ".response"
	pbRs := b.transporter.Subscribe(channelResponseTransporter)
	pubsubRs := pbRs.(*redis.PubSub)
	go b.transporter.Receive(func(cn string, data interface{}, err error) {
		go func() {
			if err != nil {
				return
			}
			dRs := ResponseTranferData{}
			mapstructure.Decode(data, &dRs)
			channelResponseTransporter := GO_SERVICE_PREFIX + "." + b.Config.NodeId + ".response." + dRs.ResponseId
			b.bus.Publish(channelResponseTransporter, dRs)
		}()
	}, pubsubRs)

}

func (b *Broker) listenActionCall(serviceName string, action Action) {
	channel := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + serviceName + "." + action.Name
	b.bus.Subscribe(channel, func(data RequestTranferData) {
		go func() {

			responseId := data.ResponseId
			ctx := Context{
				RequestId:         uuid.New().String(),
				ResponseId:        uuid.New().String(),
				Params:            data.Params,
				Meta:              data.Meta,
				FromNode:          data.CallerNodeId,
				FromService:       data.CallerService,
				FromEvent:         data.CallToEvent,
				FromAction:        data.CallerAction,
				CallingLevel:      data.CallingLevel,
				TraceParentId:     data.TraceParentId,
				TraceParentRootId: data.TraceRootParentId,
			}

			// start trace
			spanId := b.startTraceSpan("Action `"+serviceName+"."+action.Name+"`", "action", serviceName, action.Name, data.Params, ctx.FromNode, ctx.TraceParentId, ctx.CallingLevel, data.TraceRootParentId)

			ctx.TraceParentId = spanId
			ctx.Call = func(a string, params interface{}, meta interface{}) (interface{}, error) {
				ctxCall := Context{
					RequestId:         uuid.New().String(),
					ResponseId:        uuid.New().String(),
					Params:            params,
					Meta:              meta,
					FromNode:          b.Config.NodeId,
					FromService:       serviceName,
					FromAction:        action.Name,
					CallingLevel:      data.CallingLevel,
					TraceParentId:     spanId,
					TraceParentRootId: data.TraceRootParentId,
				}
				callResult, err := b.callActionOrEvent(ctxCall, a, params, meta, serviceName, action.Name, "")
				b.addTraceSpans(callResult.TraceSpans)
				if err != nil {
					return nil, err
				}
				return callResult.Data, err
			}

			// handle action
			res, e := action.Handle(&ctx)

			// end trace
			b.endTraceSpan(spanId, e)

			// response result
			responseTranferData := ResponseTranferData{
				ResponseId:      responseId,
				ResponseNodeId:  b.Config.NodeId,
				ResponseService: serviceName,
				ResponseAction:  action.Name,
				ResponseTime:    time.Now().UnixNano(),
			}

			// response trace if the exporter is console
			if b.Config.TraceConfig.TraceExpoter == TraceExporterConsole {
				trans, errFind := b.findSpan(spanId)
				if errFind == nil {
					responseTranferData.TraceSpans = append(responseTranferData.TraceSpans, *trans)
					traceSpans := b.findTraceChildrensDeep(trans.TraceId)
					for _, s := range traceSpans {
						// if s.Tags.CallerNodeId != data.CallerNodeId {
						responseTranferData.TraceSpans = append(responseTranferData.TraceSpans, *s)
						// }
					}
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

			responseChanel := GO_SERVICE_PREFIX + "." + b.Config.NodeId + ".response." + responseId
			b.bus.Publish(responseChanel, responseTranferData)
		}()
	})
}
func (b *Broker) listenEventCall(serviceName string, event Event) {
	channel := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + serviceName + "." + event.Name
	b.bus.Subscribe(channel, func(data RequestTranferData) {
		go func() {
			ctx := Context{
				RequestId:         uuid.New().String(),
				ResponseId:        uuid.New().String(),
				Params:            data.Params,
				Meta:              data.Meta,
				FromNode:          data.CallerNodeId,
				FromService:       data.CallerService,
				FromEvent:         data.CallToEvent,
				FromAction:        data.CallerAction,
				CallingLevel:      data.CallingLevel,
				TraceParentId:     data.TraceParentId,
				TraceParentRootId: data.TraceRootParentId,
			}

			// start trace
			spanId := b.startTraceSpan("Event `"+event.Name+"`", "action", serviceName, event.Name, data.Params, ctx.FromNode, ctx.TraceParentId, ctx.CallingLevel, data.TraceRootParentId)

			ctx.TraceParentId = spanId
			ctx.Call = func(a string, params interface{}, meta interface{}) (interface{}, error) {
				ctxCall := Context{
					RequestId:         uuid.New().String(),
					ResponseId:        uuid.New().String(),
					Params:            params,
					Meta:              meta,
					FromNode:          b.Config.NodeId,
					FromService:       serviceName,
					FromEvent:         event.Name,
					CallingLevel:      data.CallingLevel,
					TraceParentId:     spanId,
					TraceParentRootId: data.TraceRootParentId,
				}
				callResult, err := b.callActionOrEvent(ctxCall, a, params, meta, serviceName, "", event.Name)
				b.addTraceSpans(callResult.TraceSpans)
				if err != nil {
					return nil, err
				}
				return callResult.Data, err
			}

			// handle action
			event.Handle(&ctx)

			// end trace
			b.endTraceSpan(spanId, nil)
		}()
	})
}

// calling
func (b *Broker) callActionOrEvent(ctx Context, actionName string, params interface{}, meta interface{}, callerService string, callerAction string, callerEvent string) (ResponseTranferData, error) {
	// chose node call
	service, action, events := b.balancingRoundRobin(actionName)
	if service.Name == "" && action.Name == "" && len(events) == 0 {
		return ResponseTranferData{}, errors.New("Action or event `" + actionName + "` is not existed")
	}
	// call event
	if len(events) != 0 {
		for i := 0; i < len(events); i++ {
			for j := 0; j < len(events[i].Events); j++ {
				// Init data send
				channelTransporter := GO_SERVICE_PREFIX + "." + events[i].Node.NodeId + ".request"
				responseId := uuid.New().String()
				dataSend := RequestTranferData{
					Params:            params,
					Meta:              meta,
					RequestId:         ctx.RequestId,
					ResponseId:        responseId,
					CallerNodeId:      b.Config.NodeId,
					CallerService:     callerService,
					CallerAction:      callerAction,
					CallerEvent:       callerEvent,
					CallingLevel:      ctx.CallingLevel + 1,
					CalledTime:        time.Now().UnixNano(),
					CallToService:     events[i].Name,
					CallToEvent:       events[i].Events[j].Name,
					TraceParentId:     ctx.TraceParentId,
					TraceRootParentId: ctx.TraceParentRootId,
				}
				// push transporter
				b.transporter.Emit(channelTransporter, dataSend)
			}
		}
		return ResponseTranferData{}, nil
	}
	channelInternal := ""
	data := make(chan ResponseTranferData, 1)
	var err error
	// call action
	go func() {
		// Init data send
		channelTransporter := GO_SERVICE_PREFIX + "." + service.Node.NodeId + ".request"
		responseId := uuid.New().String()
		channelInternal = GO_SERVICE_PREFIX + "." + b.Config.NodeId + ".response." + responseId
		dataSend := RequestTranferData{
			Params:            params,
			Meta:              meta,
			RequestId:         ctx.RequestId,
			ResponseId:        responseId,
			CallerNodeId:      b.Config.NodeId,
			CallerService:     callerService,
			CallerAction:      callerAction,
			CallerEvent:       callerEvent,
			CallingLevel:      ctx.CallingLevel + 1,
			CalledTime:        time.Now().UnixNano(),
			CallToService:     service.Name,
			CallToAction:      action.Name,
			TraceParentId:     ctx.TraceParentId,
			TraceRootParentId: ctx.TraceParentRootId,
		}
		// Subscribe response data
		b.bus.Subscribe(channelInternal, func(d interface{}) {
			go func() {
				dT := ResponseTranferData{}
				mapstructure.Decode(d, &dT)
				data <- dT
				b.bus.UnSubscribe(channelInternal)
			}()
		})

		// push transporter
		b.transporter.Emit(channelTransporter, dataSend)

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
		// fmt.Println("Traces response: ", res.TraceSpans)
		b.addTraceSpans(res.TraceSpans)
		return res, nil
	case <-time.After(time.Duration(b.Config.RequestTimeOut) * time.Millisecond):
		if channelInternal != "" {
			b.bus.UnSubscribe(channelInternal)
		}
		err := errors.New("Timeout")
		return ResponseTranferData{}, err
	}
}

func (b *Broker) emitWithTimeout(channelCall string, channelReceive string, dataSend interface{}) (interface{}, error) {
	if channelReceive != "" {
		data := make(chan interface{}, 1)
		go func() {

			// Subscribe response data
			b.bus.Subscribe(channelReceive, func(d interface{}) {
				data <- d
				b.bus.UnSubscribe(channelReceive)
			})

			b.bus.Publish(channelCall, dataSend)
		}()
		select {
		case res := <-data:
			return res, nil
		case <-time.After(time.Duration(b.Config.RequestTimeOut) * time.Millisecond):
			b.bus.UnSubscribe(channelReceive)
			return nil, errors.New("Timeout")
		}
	} else {
		b.bus.Publish(channelCall, dataSend)
		return nil, nil
	}
}
