package utils

import (
	"log"
	"os"
)

// Logger 简单的日志包装器
type Logger struct {
	*log.Logger
}

// NewLogger 创建新日志实例
func NewLogger(prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lmicroseconds),
	}
}

// Info 信息日志
func (l *Logger) Info(v ...interface{}) {
	l.Logger.Println("[INFO]", v)
}

// Error 错误日志
func (l *Logger) Error(v ...interface{}) {
	l.Logger.Println("[ERROR]", v)
}

// Fatal 致命错误
func (l *Logger) Fatal(v ...interface{}) {
	l.Logger.Fatal("[FATAL]", v)
}

// DefaultLogger 默认日志实例
var DefaultLogger = NewLogger("[Kafka]")
