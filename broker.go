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
	Logger            string
	Metrics           string
	TraceConfig       TraceConfig
	DiscoveryConfig   DiscoveryConfig
	RequestTimeOut    int
}

type Broker struct {
	Config   BrokerConfig
	Services []*Service
	Started  func(*Context)
	Stoped   func(*Context)
}

var broker Broker

var debouncedEmitInfo func(f func())

func Init(config BrokerConfig) {
	initMetrics()
	broker = Broker{
		Config: config,
	}
	initDiscovery()
	initTransporter()
	initTrace()
	debouncedEmitInfo = debounce.New(1000 * time.Millisecond)
}

func LoadService(service Service) {

	broker.Services = append(broker.Services, &service)

	// add service to registry
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
		Node:    registryNode,
		Name:    service.Name,
		Actions: registryActions,
		Events:  registryEvents,
	})

	// emit info service
	debouncedEmitInfo(startDiscovery)
	// handle logic service

	// service lifecycle

	if service.Started != nil {
		go func() {
			//trace
			spanId := startTraceSpan("Service `"+service.Name+"` started", "action", "", "", map[string]interface{}{}, "", "", 1, "")
			// started

			context := Context{
				RequestId:    uuid.New().String(),
				Params:       map[string]interface{}{},
				Meta:         map[string]interface{}{},
				FromService:  "",
				FromNode:     broker.Config.NodeId,
				CallingLevel: 1,
			}
			context.Call = func(action string, params interface{}, meta interface{}) (interface{}, error) {
				callResult, err := callAction(context, action, params, meta, "", "", spanId)
				if err != nil {
					return nil, err
				}
				return callResult.Data, err
			}
			service.Started(&context)
			endTraceSpan(spanId, nil)
		}()
	}

	// actions handle
	for _, a := range service.Actions {
		go listenActionCall(service.Name, a)
	}

	// events handle
}

func Hold() {
	select {}
}
