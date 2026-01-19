package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{}) // 设置 format text

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)
}

func main() {

	log.WithFields(log.Fields{
		"animal": "dog",
		"size":   10,
	}).Info("a group of dog emerges from the zoon")
}

//输出：
//time="2021-11-11T20:51:27+08:00" level=info msg="a group of dog emerges from the zoon" animal=dog size=10
