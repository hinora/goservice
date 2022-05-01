package goservice

import (
	"expvar"

	"github.com/zserge/metric"
)

func (b *Broker) balancingRoundRobin(name string) (RegistryService, RegistryAction, []RegistryService) {
	var rs RegistryService
	var ra RegistryAction
	var re []RegistryService
	var minCall float64 = 0
	var actions []RegistryAction
	var services []RegistryService
	for _, s := range b.registryServices {
		for _, a := range s.Actions {
			if name == s.Name+"."+a.Name {
				actions = append(actions, a)
				services = append(services, s)
			}
		}
		for _, e := range s.Events {
			if name == e.Name {
				re = append(re, s)
			}
		}
	}
	if len(actions) != 0 {
		ra = actions[0]
		rs = services[0]
		for i, a := range actions {
			nameCheck := MCountCall + "." + services[i].Node.NodeId + "." + services[i].Name + "." + a.Name
			// if expvar.Get(nameCheck) == nil {
			// 	expvar.Publish(nameCheck, metric.NewCounter(MCountCallTime))
			// }

			countCheck := MetricsGetValueCounter(expvar.Get(nameCheck).(metric.Metric))
			if countCheck <= minCall {
				minCall = countCheck
				ra = a
				rs = services[i]
			}
		}
		if rs.Name != "" && ra.Name != "" {
			nameCheck := MCountCall + "." + rs.Node.NodeId + "." + rs.Name + "." + ra.Name
			expvar.Get(nameCheck).(metric.Metric).Add(1)
		}
	} else if len(re) != 0 {
		for i := 0; i < len(re); i++ {
			re[i].Actions = []RegistryAction{}
			var events []RegistryEvent
			for _, e := range re[i].Events {
				if e.Name == name {
					events = append(events, e)
				}
			}
			re[i].Events = events
		}
	}
	return rs, ra, re
}
