package main

import (
	log "github.com/sirupsen/logrus"

	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	logger := &lumberjack.Logger{
		Filename:   "./testlogrus.log",
		MaxSize:    500,  // 日志文件大小，单位是 MB
		MaxBackups: 3,    // 最大过期日志保留个数
		MaxAge:     28,   // 保留过期文件最大时间，单位 天
		Compress:   true, // 是否压缩日志，默认是不压缩。这里设置为true，压缩日志
	}

	log.SetOutput(logger) // logrus 设置日志的输出方式

}
