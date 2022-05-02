package goservice

import (
	"time"

	"github.com/google/uuid"
)

type Gateway struct {
}

func (g *Gateway) Init() Service {

	return Service{}
}

const (
	eventServiceInfo = "service.info"
)

func (b *Broker) emitServiceInfoInternal() {
	spanId := b.startTraceSpan("Emit info services", "action", "", "", map[string]interface{}{}, "", "", 1)
	for i := 0; i < len(b.registryServices); i++ {
		if b.registryServices[i].Node.NodeId != b.Config.NodeId {
			continue
		}
		for j := 0; j < len(b.registryServices[i].Events); j++ {
			if b.registryServices[i].Events[j].Name == eventServiceInfo {
				dT := RequestTranferData{
					Params:            b.registryServices,
					Meta:              nil,
					RequestId:         uuid.New().String(),
					ResponseId:        uuid.New().String(),
					CallerNodeId:      b.Config.NodeId,
					CallerService:     "",
					CallerAction:      "",
					CallerEvent:       "",
					CallingLevel:      1,
					CalledTime:        time.Now().UnixNano(),
					CallToService:     b.registryServices[i].Name,
					CallToEvent:       eventServiceInfo,
					TraceParentId:     spanId,
					TraceRootParentId: spanId,
				}
				channelCall := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + b.registryServices[i].Name + "." + eventServiceInfo
				b.emitWithTimeout(channelCall, "", dT)
			}
		}
	}
	b.endTraceSpan(spanId, nil)
}
