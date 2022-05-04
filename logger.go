package goservice

import (
	"time"
)

// LOGGER

type LogType int

const (
	LogTypeInfo LogType = iota + 1
	LogTypeWarning
	LogTypeError
)

type LogData struct {
	Type    LogType
	Message string
	Payload map[string]interface{}
	Time    int
}

type LogExternal int

const (
	LogConsole LogExternal = iota + 1
	LogFile
)

type Logconfig struct {
	Enable bool
	Type   LogExternal
}

type Log struct {
	Config  Logconfig
	Extenal interface{}
}

func (b *Broker) initLog() {
	switch b.Config.LoggerConfig.Type {
	case LogConsole:
		logExternal := LoggerConsole{
			data: []LogData{},
		}
		go logExternal.Start()
		b.logs = Log{
			Config:  b.Config.LoggerConfig,
			Extenal: &logExternal,
		}
		break
	case LogFile:
		break
	}
}

// extenal console
func (b *Broker) LogInfo(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	log.Type = LogTypeInfo
	b.logs.exportLog(log)
}
func (b *Broker) LogWarning(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	log.Type = LogTypeWarning
	b.logs.exportLog(log)
}
func (b *Broker) LogError(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	log.Type = LogTypeError
	b.logs.exportLog(log)
}
func (l *Log) exportLog(log LogData) {
	if !l.Config.Enable {
		return
	}
	switch l.Config.Type {
	case LogConsole:
		logExternal := l.Extenal.(*LoggerConsole)
		logExternal.WriteLog(log)
		break
	case LogFile:
		break
	}
}
