package goservice

import (
	"time"

	"github.com/fatih/color"
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
	Console LogExternal = iota + 1
	File
)

type Logconfig struct {
	Enable bool
	Type   LogExternal
}

func logInfo(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	// just demo on console. Need replace with log external
	color.New(color.FgCyan).Add(color.Underline).Print(time.Unix(int64(log.Time), 0))
	color.New(color.FgGreen).Add(color.Underline).Print(" Info ")
	color.New(color.FgCyan).Add(color.Underline).Print(log.Message)
	color.New(color.FgBlack).Add(color.Underline).Println(" ")
}
func logWarning(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	// just demo on console. Need replace with log external
	color.New(color.FgCyan).Add(color.Underline).Print(time.Unix(int64(log.Time), 0))
	color.New(color.FgYellow).Add(color.Underline).Print(" Waring ")
	color.New(color.FgCyan).Add(color.Underline).Print(log.Message)
	color.New(color.FgBlack).Add(color.Underline).Println(" ")
}
func logError(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	// just demo on console. Need replace with log external
	color.New(color.FgCyan).Add(color.Underline).Print(time.Unix(int64(log.Time), 0))
	color.New(color.FgRed).Add(color.Underline).Print(" Error ")
	color.New(color.FgCyan).Add(color.Underline).Print(log.Message)
	color.New(color.FgBlack).Add(color.Underline).Println(" ")
}
