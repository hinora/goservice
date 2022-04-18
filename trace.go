package goservice

import (
	"errors"
	"fmt"
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

const (
	TraceExporterConsole TraceExpoter = iota + 1
	TraceExporterDDDog
)

var traceSpans map[string]*traceSpan

func initTrace() {
	traceSpans = map[string]*traceSpan{}
}

func addTraceSpan(span *traceSpan) string {
	traceSpans[span.TraceId] = span
	return span.TraceId
}
func addTraceSpans(spans []traceSpan) {
	for _, s := range spans {
		traceSpans[s.TraceId] = &s
	}
}

func startTraceSpan(name string, types string, service string, action string, params interface{}, callerNodeId string, parentId string, callingLevel int, requestId ...string) string {
	id := uuid.New().String()
	remoteCall := false
	if callerNodeId != "" && broker.Config.NodeId != callerNodeId {
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
			NodeId:       broker.Config.NodeId,
			RequestId:    reqId,
			Params:       params,
			FromCache:    false,
		},
	}
	addTraceSpan(&trace)
	return id
}

func endTraceSpan(spanId string, err error) {
	span, e := findSpan(spanId)
	if e != nil {
		return
	}
	span.FinishTime = time.Now().UnixNano()
	span.Duration = span.FinishTime - span.StartTime
	if span.Error == nil && err != nil {
		span.Error = err
	}

	spans := findTraceChildrens(span.Tags.RequestId)
	for _, s := range spans {
		fmt.Println("Trace: ", *s)
	}
	// export trace span to trace exporter
	// go s.Trace.PrintSpan(spanId)
}
func findSpan(spanId string) (*traceSpan, error) {
	if value, ok := traceSpans[spanId]; ok {
		return value, nil
	}
	return &traceSpan{}, errors.New("Span not exist")
}

func findTraceChildrens(requestID string) []*traceSpan {
	traces := []*traceSpan{}
	for k, v := range traceSpans {
		if v.Tags.RequestId == requestID {
			traces = append(traces, traceSpans[k])
		}
	}
	return traces
}
func findTraceChildrensDeep(spanId string) []*traceSpan {
	traces := []*traceSpan{}
	for k, v := range traceSpans {
		if v.ParentId == spanId {
			traces = append(traces, traceSpans[k])
		}
	}
	if len(traces) != 0 {
		for _, t := range traces {
			traces = append(traces, findTraceChildrensDeep(t.TraceId)...)
		}
	}
	return traces
}

func removeSpan(spanId string) {
	delete(traceSpans, spanId)
}
func removeSpanByParent(parentId string) {
	for _, v := range traceSpans {
		if v.ParentId == parentId {
			delete(traceSpans, v.TraceId)
		}
	}
}

// interface for exporter
