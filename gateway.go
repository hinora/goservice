package goservice

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strconv"
	"time"

	"github.com/bep/debounce"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GatewayConfigRoute struct {
	Path             string
	WhileList        []string
	StaticPath       string
	StaticFolderRoot string
}
type GatewayConfig struct {
	Name   string
	Host   string
	Port   int
	Routes []GatewayConfigRoute
}

type Gateway struct {
	Gin          *gin.Engine
	GinRoutes    []*gin.RouterGroup
	Services     []RegistryService
	debouncedGen func(f func())
	Config       GatewayConfig
	Service      *Service
}

func (g *Gateway) MapServices() {
	for _, r := range g.Config.Routes {
		g.Service.Broker.LogInfo("Generate route: `" + r.Path + "`")
		rG := g.Gin.Group(r.Path)
		for _, s := range g.Services {
			for _, a := range s.Actions {
				if g.checkMatch(s.Name, a, r) {
					g.genHandle(rG, s.Name, a)
				}
			}
		}
		if r.StaticPath != "" && r.StaticFolderRoot != "" {
			g.Service.Broker.LogInfo("Generate route static folder: `" + r.Path + r.StaticPath + "`")
			rG.Static(r.StaticPath, r.StaticFolderRoot)
		}
		g.GinRoutes = append(g.GinRoutes, rG)
	}
	g.Gin.Run(g.Config.Host + ":" + strconv.Itoa(g.Config.Port))
}

func (g *Gateway) genHandle(r *gin.RouterGroup, serviceName string, action RegistryAction) {
	pathMapping := serviceName + action.Rest.Path
	g.Service.Broker.LogInfo("Generate gateway end point: `" + action.Rest.Method.String() + "` " + "/" + pathMapping)
	handle := func(c *gin.Context) {
		params := g.parseParam(c)
		data, err := g.Service.Broker.Call(g.Service.Name, "`"+action.Rest.Method.String()+"` "+"/"+pathMapping, serviceName+"."+action.Name, params, CallOpts{})
		if err != nil {
			c.JSON(500, err.Error())
		} else {
			c.JSON(200, data)
		}
	}
	switch action.Rest.Method {
	case GET:
		r.GET(pathMapping, handle)
		break
	case POST:
		r.POST(pathMapping, handle)
		break
	case PUT:
		r.PUT(pathMapping, handle)
		break
	case DELETE:
		r.DELETE(pathMapping, handle)
		break
	case PATCH:
		r.PATCH(pathMapping, handle)
		break
	case HEAD:
		r.HEAD(pathMapping, handle)
		break
	case OPTIONS:
		r.OPTIONS(pathMapping, handle)
		break
	}
}
func (g *Gateway) checkMatch(serviceName string, action RegistryAction, r GatewayConfigRoute) bool {
	if action.Rest == (Rest{}) {
		return false
	}
	for _, wL := range r.WhileList {
		check, err := regexp.MatchString(wL, serviceName+"."+action.Name)
		if check && err == nil {
			return true
		}
	}
	return false
}

func (g *Gateway) parseParam(c *gin.Context) map[string]interface{} {
	// query
	queryRaw := c.Request.URL.Query()
	query := map[string]interface{}{}
	for k, v := range queryRaw {
		if len(v) == 1 {
			query[k] = v[0]
		} else {
			query[k] = v
		}
	}

	// param
	paramsRaw := c.Params
	params := map[string]interface{}{}
	for i := 0; i < len(paramsRaw); i++ {
		params[paramsRaw[i].Key] = paramsRaw[i].Value
	}

	// body
	jsonDataByte, _ := ioutil.ReadAll(c.Request.Body)
	var jsonData map[string]interface{}
	json.Unmarshal(jsonDataByte, &jsonData)
	return mergeMap(query, params, jsonData)
}
func mergeMap(m ...map[string]interface{}) map[string]interface{} {
	if len(m) == 1 {
		return m[0]
	}
	result := m[0]
	for i := 1; i < len(m); i++ {
		for k, v := range m[i] {
			result[k] = v
		}
	}
	return result
}
func InitGateway(config GatewayConfig) *Service {
	gin.SetMode(gin.ReleaseMode)
	gateway := Gateway{
		debouncedGen: debounce.New(1000 * time.Millisecond),
		Gin:          gin.New(),
		Config:       config,
	}
	service := Service{
		Name: config.Name,
		Events: []Event{
			{
				Name: eventServiceInfo,
				Handle: func(ctx *Context) {
					data := ctx.Params.([]RegistryService)
					gateway.Services = data
					gateway.Gin = gin.New()
					gateway.debouncedGen(gateway.MapServices)
				},
			},
		},
	}
	gateway.Service = &service
	return &service
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
