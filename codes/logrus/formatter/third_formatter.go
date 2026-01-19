package main

import (
	"net/http"
	"time"

	joonix "github.com/joonix/log"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(joonix.NewFormatter())
	log.Info("hello world")

	log.WithField("httpRequest", &joonix.HTTPRequest{
		Request:      r,
		Status:       http.StatusOK,
		ResponseSize: 3211,
		Latency:      123 * time.Millisecond,
	}).Info("additional info")
}
