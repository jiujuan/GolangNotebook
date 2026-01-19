package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{}) // 设置 format json
	// log.SetLevel(log.WarnLevel)            // 设置输出警告级别

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)
}

func main() {

	log.WithFields(log.Fields{
		"animal": "dog",
		"size":   10,
	}).Info("a group of dog emerges from the zoon")

	log.WithFields(log.Fields{
		"omg":    true,
		"number": 12,
	}).Warn("the group's number increased")

	log.WithFields(log.Fields{
		"omg":    true,
		"number": 100,
	}).Fatal("th ice breaks")

	// the logrus.Entry returned from WithFields()
	contextLogger := log.WithFields(log.Fields{
		"common": "this is a common filed",
		"other":  "i also should be logged always",
	})

	contextLogger.Info("I'll be logged with common and other field")
	contextLogger.Info("Me too")

}
