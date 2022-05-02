package goservice

import (
	"reflect"
	"sync"
)

type EventBusData struct {
	Channel string
	Handle  []reflect.Value
}

type EventBus struct {
	data map[string]*EventBusData
	lock sync.RWMutex
}

func initEventBus() EventBus {
	return EventBus{
		lock: sync.RWMutex{},
		data: make(map[string]*EventBusData),
	}
}
func (eb *EventBus) Subscribe(channel string, fn interface{}) {
	eb.lock.Lock()
	defer eb.lock.Unlock()
	if value, ok := eb.data[channel]; ok {
		value.Handle = append(value.Handle, reflect.ValueOf(fn))
	} else {
		eb.data[channel] = &EventBusData{
			Channel: channel,
			Handle: []reflect.Value{
				reflect.ValueOf(fn),
			},
		}
	}
}

func (eb *EventBus) Publish(channel string, data interface{}) {
	eb.lock.RLock()
	defer eb.lock.RUnlock()
	if value, ok := eb.data[channel]; ok {
		for i := 0; i < len(value.Handle); i++ {
			h := value.Handle[i]
			go func() {
				h.Call([]reflect.Value{
					reflect.ValueOf(data),
				})
			}()
		}
	}
}
func (eb *EventBus) UnSubscribe(channel string) {
	eb.lock.Lock()
	defer eb.lock.Unlock()
	if value, ok := eb.data[channel]; ok {
		delete(eb.data, value.Channel)
	}
}
