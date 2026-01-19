package main

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

type MyJSONFormatter struct {
	JSONPrefix string
	Otherdata  string
}

func (my *MyJSONFormatter) Format(entry *log.Entry) ([]byte, error) {
	// fmt.Println(entry.Data["msg"])

	entry.Data["msg"] = fmt.Sprintf("%s%s", my.JSONPrefix, my.Otherdata)
	json, err := jsoniter.Marshal(&entry.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON , %w", err)
	}
	return append(json, '\n'), nil

}

func main() {
	formatter := &MyJSONFormatter{
		JSONPrefix: "jsonprefix-",
		Otherdata:  ":otherdata:",
	}

	log.SetFormatter(formatter)
	log.Info("this is customered formatter")
}
