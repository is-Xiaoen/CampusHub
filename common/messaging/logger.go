package messaging

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
)

// watermillLogger Watermill 日志适配器
type watermillLogger struct {
	serviceName string
}

// newWatermillLogger 创建 Watermill 日志适配器
func newWatermillLogger(serviceName string) watermill.LoggerAdapter {
	return &watermillLogger{
		serviceName: serviceName,
	}
}

func (l *watermillLogger) Error(msg string, err error, fields watermill.LogFields) {
	fmt.Printf("[ERROR] [%s] %s: %v %v\n", l.serviceName, msg, err, fields)
}

func (l *watermillLogger) Info(msg string, fields watermill.LogFields) {
	fmt.Printf("[INFO] [%s] %s %v\n", l.serviceName, msg, fields)
}

func (l *watermillLogger) Debug(msg string, fields watermill.LogFields) {
	fmt.Printf("[DEBUG] [%s] %s %v\n", l.serviceName, msg, fields)
}

func (l *watermillLogger) Trace(msg string, fields watermill.LogFields) {
	fmt.Printf("[TRACE] [%s] %s %v\n", l.serviceName, msg, fields)
}

func (l *watermillLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	// 简单实现，返回自身
	return l
}
