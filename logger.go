package goservice

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
