package main

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func main() {
	uid := uuid.New()
	request_id := uid
	trace_id := uid
	requestLogger := log.WithFields(log.Fields{"request_id": request_id, "trace_id": trace_id})
	requestLogger.Info("something happened on that request")
	requestLogger.Warn("something not great happened")
}
