package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	log.RegisterExitHandler(func() {
		fmt.Println("发生了fatal异常，执行一些必要的处理工作")
	})

	log.Warn("warn")
	log.Fatal("fatal")
	log.Info("info") //不会执行
}
