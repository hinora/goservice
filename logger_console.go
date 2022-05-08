package goservice

import (
	"time"

	"github.com/fatih/color"
)

type LoggerConsole struct {
	data []LogData
}

func (l *LoggerConsole) WriteLog(log LogData) {
	l.data = append(l.data, log)
}
func (l *LoggerConsole) Start() {
	l.exportLog()
}
func (l *LoggerConsole) exportLog() {
	if len(l.data) != 0 {
		log := l.data[0]
		switch log.Type {
		case LogTypeInfo:
			l.logInfo(log)
			break
		case LogTypeWarning:
			l.logWarning(log)
			break
		case LogTypeError:
			l.logError(log)
			break
		}
		l.data = l.data[1:]
	} else {
		time.Sleep(time.Millisecond * 1)
	}
	l.exportLog()
}
func (l *LoggerConsole) logInfo(log LogData) {
	color.New(color.FgCyan).Print(time.Unix(int64(log.Time), 0))
	color.New(color.FgGreen).Print(" Info ")
	color.New(color.FgCyan).Print(log.Message)
	color.New(color.FgBlack).Println(" ")
}
func (l *LoggerConsole) logWarning(log LogData) {
	color.New(color.FgCyan).Print(time.Unix(int64(log.Time), 0))
	color.New(color.FgYellow).Print(" Waring ")
	color.New(color.FgCyan).Print(log.Message)
	color.New(color.FgBlack).Println(" ")
}
func (l *LoggerConsole) logError(log LogData) {
	color.New(color.FgCyan).Print(time.Unix(int64(log.Time), 0))
	color.New(color.FgRed).Print(" Error ")
	color.New(color.FgCyan).Print(log.Message)
	color.New(color.FgBlack).Println(" ")
}
