package goservice

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type traceSpan struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	TraceId    string        `json:"traceID"`
	ParentId   string        `json:"parentID"`
	Service    string        `json:"service"`
	StartTime  int64         `json:"starttime"`
	FinishTime int64         `json:"finishtime"`
	Duration   int64         `json:"duration"`
	Error      interface{}   `json:"error"`
	Logs       []interface{} `json:"logs"`
	Tags       tags          `json:"tags"`
}
type tags struct {
	CallingLevel int                    `json:"callingLevel"`
	Action       string                 `json:"action"`
	RemoteCall   bool                   `json:"remoteCall"`
	CallerNodeId string                 `json:"callerNodeID"`
	NodeId       string                 `json:"nodeID"`
	Options      map[string]interface{} `json:"options"`
	RequestId    string                 `json:"requestID"`
	Params       interface{}            `json:"params"`
	FromCache    bool                   `json:"fromCache"`
}

type TraceConfig struct {
	Enabled             bool
	WindowTime          int
	TraceExpoter        TraceExpoter
	TraceExporterConfig interface{}
}

type TraceExpoter int

type TraceConsoleConfig struct {
}

type Trace struct {
	Exporter interface{}
	Lock     sync.RWMutex
}

const (
	TraceExporterConsole TraceExpoter = iota + 1
	TraceExporterDDDog
)

func (b *Broker) initTrace() {
	logInfo("Trace start")
	b.traceSpans = map[string]*traceSpan{}
	if !b.Config.TraceConfig.Enabled {
		return
	}
	switch b.Config.TraceConfig.TraceExpoter {
	case TraceExporterConsole:
		b.trace = Trace{
			Exporter: initTraceConsole(b),
			Lock:     sync.RWMutex{},
		}
	}
}

func (b *Broker) addTraceSpan(span *traceSpan) string {
	if !b.Config.TraceConfig.Enabled {
		return ""
	}
	b.trace.Lock.Lock()
	defer b.trace.Lock.Unlock()
	b.traceSpans[span.TraceId] = span
	return span.TraceId
}
func (b *Broker) addTraceSpans(spans []traceSpan) {
	if !b.Config.TraceConfig.Enabled {
		return
	}
	b.trace.Lock.Lock()
	defer b.trace.Lock.Unlock()
	for i := 0; i < len(spans); i++ {
		b.traceSpans[spans[i].TraceId] = &spans[i]
	}
}

func (b *Broker) startTraceSpan(name string, types string, service string, action string, params interface{}, callerNodeId string, parentId string, callingLevel int, requestId ...string) string {
	if !b.Config.TraceConfig.Enabled {
		return ""
	}
	id := uuid.New().String()
	remoteCall := false
	if callerNodeId != "" && b.Config.NodeId != callerNodeId {
		remoteCall = true
	}
	reqId := id
	if len(requestId) != 0 {
		reqId = requestId[0]
	}
	trace := traceSpan{
		Name:       name,
		Type:       types,
		TraceId:    id,
		ParentId:   parentId,
		Service:    service,
		StartTime:  time.Now().UnixNano(),
		FinishTime: 0,
		Duration:   0,
		Tags: tags{
			CallingLevel: callingLevel,
			Action:       action,
			RemoteCall:   remoteCall,
			CallerNodeId: callerNodeId,
			NodeId:       b.Config.NodeId,
			RequestId:    reqId,
			Params:       params,
			FromCache:    false,
		},
	}
	b.addTraceSpan(&trace)
	return id
}

func (b *Broker) endTraceSpan(spanId string, err error) {
	if !b.Config.TraceConfig.Enabled {
		return
	}
	span, e := b.findSpan(spanId)
	if e != nil {
		return
	}
	span.FinishTime = time.Now().UnixNano()
	span.Duration = span.FinishTime - span.StartTime
	if span.Error == nil && err != nil {
		span.Error = err
	}

	switch b.Config.TraceConfig.TraceExpoter {
	case TraceExporterConsole:
		if span.TraceId == span.Tags.RequestId {
			spanChild := b.findTraceChildrens(span.Tags.RequestId)
			ex := b.trace.Exporter.(*traceConsole)
			ex.ExportSpan(spanChild)
		}
	}
}
func (b *Broker) findSpan(spanId string) (*traceSpan, error) {
	b.trace.Lock.RLock()
	defer b.trace.Lock.RUnlock()
	if value, ok := b.traceSpans[spanId]; ok {
		return value, nil
	}
	return &traceSpan{}, errors.New("Span not exist")
}

func (b *Broker) findTraceChildrens(requestID string) []*traceSpan {
	traces := []*traceSpan{}
	b.trace.Lock.RLock()
	defer b.trace.Lock.RUnlock()
	for k, v := range b.traceSpans {
		if v.Tags.RequestId == requestID {
			traces = append(traces, b.traceSpans[k])
		}
	}
	return traces
}
func (b *Broker) findTraceChildrensDeep(spanId string) []*traceSpan {
	traces := []*traceSpan{}
	b.trace.Lock.RLock()
	defer b.trace.Lock.RUnlock()
	for _, v := range b.traceSpans {
		if v.ParentId == spanId {
			traces = append(traces, v)
		}
	}
	if len(traces) != 0 {
		for _, t := range traces {
			traces = append(traces, b.findTraceChildrensDeep(t.TraceId)...)
		}
	}
	return traces
}

func (b *Broker) removeSpan(spanId string) {
	b.trace.Lock.Lock()
	defer b.trace.Lock.Unlock()
	delete(b.traceSpans, spanId)
}
func (b *Broker) removeSpanByParent(parentId string) {
	b.trace.Lock.Lock()
	defer b.trace.Lock.Unlock()
	for _, v := range b.traceSpans {
		if v.ParentId == parentId {
			delete(b.traceSpans, v.TraceId)
		}
	}
}

// interface for exporter
