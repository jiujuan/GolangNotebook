package main

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	log.Out = os.Stdout // 设置输出日志位置，可以设置日志到file里

	log.WithFields(logrus.Fields{
		"fruit": "apple",
		"size":  20,
	}).Info(" a lot of apples on the tree")
}
