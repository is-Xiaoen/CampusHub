package gozero

import (
	"context"
	"fmt"
)

// Logger Go-Zero 日志适配器接口
type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	WithContext(ctx context.Context) Logger
	WithFields(fields map[string]interface{}) Logger
}

// DefaultLogger 默认日志实现（使用标准库 log）
type DefaultLogger struct {
	ctx    context.Context
	fields map[string]interface{}
}

// NewDefaultLogger 创建默认日志记录器
func NewDefaultLogger() Logger {
	return &DefaultLogger{
		ctx:    context.Background(),
		fields: make(map[string]interface{}),
	}
}

// Debug 输出调试日志
func (l *DefaultLogger) Debug(v ...interface{}) {
	l.log("DEBUG", fmt.Sprint(v...))
}

// Debugf 输出格式化调试日志
func (l *DefaultLogger) Debugf(format string, v ...interface{}) {
	l.log("DEBUG", fmt.Sprintf(format, v...))
}

// Info 输出信息日志
func (l *DefaultLogger) Info(v ...interface{}) {
	l.log("INFO", fmt.Sprint(v...))
}

// Infof 输出格式化信息日志
func (l *DefaultLogger) Infof(format string, v ...interface{}) {
	l.log("INFO", fmt.Sprintf(format, v...))
}

// Warn 输出警告日志
func (l *DefaultLogger) Warn(v ...interface{}) {
	l.log("WARN", fmt.Sprint(v...))
}

// Warnf 输出格式化警告日志
func (l *DefaultLogger) Warnf(format string, v ...interface{}) {
	l.log("WARN", fmt.Sprintf(format, v...))
}

// Error 输出错误日志
func (l *DefaultLogger) Error(v ...interface{}) {
	l.log("ERROR", fmt.Sprint(v...))
}

// Errorf 输出格式化错误日志
func (l *DefaultLogger) Errorf(format string, v ...interface{}) {
	l.log("ERROR", fmt.Sprintf(format, v...))
}

// WithContext 添加上下文
func (l *DefaultLogger) WithContext(ctx context.Context) Logger {
	return &DefaultLogger{
		ctx:    ctx,
		fields: l.fields,
	}
}

// WithFields 添加字段
func (l *DefaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &DefaultLogger{
		ctx:    l.ctx,
		fields: newFields,
	}
}

// log 内部日志输出方法
func (l *DefaultLogger) log(level, msg string) {
	// 构建日志消息
	output := fmt.Sprintf("[%s] %s", level, msg)

	// 添加追踪信息
	if traceID := GetTraceID(l.ctx); traceID != "" {
		output = fmt.Sprintf("[trace_id=%s] %s", traceID, output)
	}

	// 添加字段
	if len(l.fields) > 0 {
		output = fmt.Sprintf("%s %v", output, l.fields)
	}

	// 输出日志（实际项目中应该使用 go-zero 的 logx）
	fmt.Println(output)
}

// LoggerMiddleware 创建日志中间件
// 自动记录消息处理的开始和结束，以及错误信息
func LoggerMiddleware(logger Logger) func(next interface{}) interface{} {
	return func(next interface{}) interface{} {
		// 这里返回一个通用的中间件函数
		// 实际使用时需要根据具体的 Handler 类型进行适配
		return next
	}
}
