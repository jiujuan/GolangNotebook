package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {
	log.WithFields(log.Fields{
		"animal": "walrus",
	}).Info("a walrus appears")
}

// 运行输出：
// time="2021-11-11T17:41:48+08:00" level=info msg="a walrus appears" animal=walrus
