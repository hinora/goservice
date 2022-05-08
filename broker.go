package goservice

import (
	"time"

	"github.com/bep/debounce"
	"github.com/google/uuid"
)

// BROKER
type BrokerConfig struct {
	NodeId            string
	TransporterConfig TransporterConfig
	LoggerConfig      Logconfig
	Metrics           string
	TraceConfig       TraceConfig
	DiscoveryConfig   DiscoveryConfig
	RequestTimeOut    int
}

type Broker struct {
	Config             BrokerConfig
	Services           []*Service
	Started            func(*Context)
	Stoped             func(*Context)
	transporter        Transporter
	bus                EventBus
	traceSpans         map[string]*traceSpan
	channelPrivateInfo string
	registryServices   []RegistryService
	registryNodes      []RegistryNode
	registryNode       RegistryNode
	debouncedEmitInfo  func(f func())
	trace              Trace
	logs               Log
}

func Init(config BrokerConfig) *Broker {
	initMetrics()
	broker := Broker{
		Config: config,
	}
	broker.initDiscovery()
	broker.initTransporter()
	broker.debouncedEmitInfo = debounce.New(1000 * time.Millisecond)
	broker.debouncedEmitInfo(broker.startDiscovery)
	broker.initTrace()
	broker.initLog()
	broker.LogInfo("Broker started")
	return &broker
}

func (b *Broker) LoadService(service *Service) {
	b.LogInfo("Load service `" + service.Name + "`")
	b.Services = append(b.Services, service)

	// add service to registry
	var registryActions []RegistryAction
	for _, a := range service.Actions {
		registryActions = append(registryActions, RegistryAction{
			Name:   a.Name,
			Params: a.Params,
			Rest:   a.Rest,
		})
	}
	var registryEvents []RegistryEvent
	for _, e := range service.Events {
		registryEvents = append(registryEvents, RegistryEvent{
			Name:   e.Name,
			Params: e.Params,
		})
	}
	b.registryServices = append(b.registryServices, RegistryService{
		Node:    b.registryNode,
		Name:    service.Name,
		Actions: registryActions,
		Events:  registryEvents,
	})

	// emit info service
	b.debouncedEmitInfo(b.startDiscovery)
	// handle logic service

	// mapping broker to service
	service.Broker = b
	// service lifecycle
	if service.Started != nil {
		go func() {
			//trace
			spanId := b.startTraceSpan("Service `"+service.Name+"` started", "action", "", "", map[string]interface{}{}, "", "", 1)
			// started

			context := Context{
				RequestId:         uuid.New().String(),
				Params:            map[string]interface{}{},
				Meta:              map[string]interface{}{},
				FromService:       "",
				FromNode:          b.Config.NodeId,
				CallingLevel:      1,
				TraceParentId:     spanId,
				TraceParentRootId: spanId,
			}
			context.Service = service
			context.Call = func(action string, params interface{}, opts ...CallOpts) (interface{}, error) {
				optsTemp := CallOpts{}
				if len(opts) != 0 {
					optsTemp = opts[0]
				}
				ctxCall := Context{
					RequestId:         uuid.New().String(),
					ResponseId:        uuid.New().String(),
					Params:            params,
					Meta:              context.Meta,
					FromNode:          b.Config.NodeId,
					FromService:       service.Name,
					FromAction:        "",
					CallingLevel:      1,
					TraceParentId:     spanId,
					TraceParentRootId: spanId,
				}
				callResult, err := b.callActionOrEvent(ctxCall, action, params, optsTemp, service.Name, "", "")
				b.addTraceSpans(callResult.TraceSpans)
				if err != nil {
					return nil, err
				}
				return callResult.Data, err
			}
			service.Started(&context)
			b.endTraceSpan(spanId, nil)
		}()
	}

	// actions handle
	for _, a := range service.Actions {
		go b.listenActionCall(service.Name, a)
	}

	// events handle
	for _, e := range service.Events {
		go b.listenEventCall(service.Name, e)
	}
}

func (b *Broker) Call(callerService string, traceName string, action string, params interface{}, opts CallOpts) (interface{}, error) {
	//trace
	if traceName == "" {
		traceName = "Call from service `" + callerService + "`"
	}
	spanId := b.startTraceSpan(traceName, "action", "", "", map[string]interface{}{}, "", "", 1)

	ctxCall := Context{
		RequestId:         uuid.New().String(),
		ResponseId:        uuid.New().String(),
		Params:            params,
		Meta:              nil,
		FromNode:          b.Config.NodeId,
		FromService:       callerService,
		FromAction:        "",
		CallingLevel:      1,
		TraceParentId:     spanId,
		TraceParentRootId: spanId,
	}
	callResult, err := b.callActionOrEvent(ctxCall, action, params, opts, "", "", "")
	b.addTraceSpans(callResult.TraceSpans)
	b.endTraceSpan(spanId, err)
	if err != nil {
		return nil, err
	}
	return callResult.Data, err
}
func (b *Broker) Hold() {
	select {}
}
