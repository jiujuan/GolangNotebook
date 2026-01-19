## 介绍 logrus

它是一个结构化、插件化的日志记录库。完全兼容 golang 标准库中的日志模块。它还内置了 2 种日志输出格式 JSONFormatter 和 TextFormatter，来定义输出的日志格式。

github地址：[https://github.com/sirupsen/logrus](https://github.com/sirupsen/logrus)

## logrus 使用

> 使用的版本：logrus v1.8.1

### 1. 开始使用

```go
package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {
	log.WithFields(log.Fields{
		"animal": "walrus",
	}).Info("a walrus appears")
}
```

运行输出：

```shell
time="2021-11-11T17:41:48+08:00" level=info msg="a walrus appears" animal=walrus
```

### 2. 设置日志格式,日志级别,输出方式

#### 设置日志格式

1）**内置日志格式**

[log Formatter](https://github.com/sirupsen/logrus#formatters)，logrus内置的formatter有 2 种，logrus.TextFormatter 和 logrus.JSONFormatter

- logrus.JSONFormatter{}， 设置为 json 格式，所有设置选项在 [logrus.JSONFormatter](https://pkg.go.dev/github.com/sirupsen/logrus#JSONFormatter)

  ```go
  log.SetFormatter(&log.JSONFormatter{
      TimestampFormat: "2006-01-02 15:04:05", // 设置json里的日期输出格式
  })
  
  log.SetFormatter(&log.JSONFormatter{}) // 设置为json格式
  ```

  

- logrus.TextFormatter{}，设置为文本格式，所有的设置选项在 [logrus.TextFormatter](https://pkg.go.dev/github.com/sirupsen/logrus#TextFormatter)

  ```go
  log.SetFormatter(&log.TextFormatter{
      TimestampFormat: "2006-01-02 15:04:05",
      ForceColors:  true,
      EnvironmentOverrideColors: true,
      // FullTimestamp:true,
      // DisableLevelTruncation:true,
  })
  ```

2）**自定义日志格式**

可以根据 [Formatter](https://pkg.go.dev/github.com/sirupsen/logrus#Formatter) 接口自定义日志格式，里面有一个 Format 方法，这个 Format 方法里有一个struct类型数据 [*Entry](https://pkg.go.dev/github.com/sirupsen/logrus#Entry)， Entry.Data 是所有字段集合，[Fields](https://pkg.go.dev/github.com/sirupsen/logrus#Fields) 类型为 map[string]interface{}。

比如：entry.Data["msg"]，entry.Data["time"]`. The timestamp

```go
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
```



3）**第三方自定义formatter设置日志格式**

- [`FluentdFormatter`](https://github.com/joonix/log). Formats entries that can be parsed by Kubernetes and Google Container Engine.
- [`logstash`](https://github.com/bshuster-repo/logrus-logstash-hook). Logs fields as [Logstash](http://logstash.net/) Events.
- [`caption-json-formatter`](https://github.com/nolleh/caption_json_formatter). logrus's message json formatter with human-readable caption added.
- [`powerful-logrus-formatter`](https://github.com/zput/zxcTool). get fileName, log's line number and the latest function's name when print log; Sava log to files.

等等

#### 设置日志级别

logrus日志一共7级别, 从高到低: panic, fatal, error, warn, info, debug, trace.

- log.SetLevel(log.WarnLevel) // 设置输出警告级别

#### 设置日志输出方式

- log.SetOutput(os.Stdout) // 输入到 Stdout，默认输出到 Stderr
- logfile, _ := os.OpenFile("./logrus.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
   logrus.SetOutput(logfile) // 输出到文件里

例子：

```go
package main

import (
	log "github.com/sirupsen/logrus"
    "os"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{}) // 设置 format json
	log.SetLevel(log.WarnLevel) // 设置输出警告级别
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
	// 共同字段输出
	contextLogger.Info("I'll be logged with common and other field")
	contextLogger.Info("Me too")
}

```

运行输出：

```shell
{"level":"warning","msg":"the group's number increased","number":12,"omg":true,"time":"2021-11-11T18:00:55+08:00"}
{"level":"fatal","msg":"th ice breaks","number":100,"omg":true,"time":"2021-11-11T18:00:55+08:00"}
```

从输出的结果看出，Info 级别的日志信息都没有输出出来。



屏蔽设置日志级别的代码

```go
func init() {
	log.SetFormatter(&log.JSONFormatter{}) // 设置 format json
	// log.SetLevel(log.WarnLevel)            // 设置输出警告级别
}
```

在运行输出：

```shell
{"animal":"dog","level":"info","msg":"a group of dog emerges from the zoon","size":10,"time":"2021-11-11T18:26:45+08:00"}
{"level":"warning","msg":"the group's number increased","number":12,"omg":true,"time":"2021-11-11T18:26:45+08:00"}
{"level":"fatal","msg":"th ice breaks","number":100,"omg":true,"time":"2021-11-11T18:26:45+08:00"}
exit status 1
```

从输出的日志信息来看，并没有输出 contextLogger 的日志info信息，日志信息没有输出，为啥没有输出日志？

把上面的 Fatal 输出日志屏蔽掉：

```go
// log.WithFields(log.Fields{
	// 	"omg":    true,
	// 	"number": 100,
	// }).Fatal("th ice breaks")
```

在运行输出：

```shell
{"animal":"dog","level":"info","msg":"a group of dog emerges from the zoon","size":10,"time":"2021-11-11T18:28:56+08:00"}
{"level":"warning","msg":"the group's number increased","number":12,"omg":true,"time":"2021-11-11T18:28:56+08:00"}
{"common":"this is a common filed","level":"info","msg":"I'll be logged with common and other field","other":"i also should be logged always","time":"2021-11-11T18:28:56+08:00"}
{"common":"this is a common filed","level":"info","msg":"Me too","other":"i also should be logged always","time":"2021-11-11T18:28:56+08:00"}
```

这时候可以输出 contextLogger 日志信息了。

### 3. logrus 的 Fatal 处理

上面的例子定义了输出 Fatal 日志后，其后的日志都不能输出了，这是为什么？日志后面有个信息 `exit status 1`，

因为 logrus 的 Fatal 输出后，会执行 os.Exit(1)。那如果程序后面还有一些必要的程序要处理怎么办？

logrus 提供了 [RegisterExitHandler](https://pkg.go.dev/github.com/sirupsen/logrus#RegisterExitHandler) 方法，在 fatal 异常时处理一些问题。

```go
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
```

运行输出：

```shell
time="2021-11-11 21:48:25" level=warning msg=warn
time="2021-11-11 21:48:25" level=fatal msg=fatal
发生了fatal异常，执行一些必要的处理工作
exit status 1
```

### 4. 切分日志文件

如果日志文件太大了，想切分成小文件，但是 logrus 没有提供这个功能。

一种是借助linux系统的 logrotate 命令来切分 logrus 生成的日志文件。

另外一种是用 logrus 的 hook 功能，做一个切分日志的插件。找到了 [file-rotatelogs](https://github.com/lestrrat-go/file-rotatelogs)，但是这个库状态

已经是 archived 状态，库作者现在不接受任何修改，他也不继续维护了。所以使用还是慎重些。

在 [logrus issue](https://github.com/sirupsen/logrus/issues/655) 里找到了这个 https://github.com/natefinch/lumberjack 切割文件的库。

例子：

```go
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
```

### 5. 设置 logrus 实例

如果一个应用有多个地方使用日志，可以单独实例化一个 logrus，作为全局的日志实例。

```go
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
```

输出：

```shell
time="2021-11-11T18:39:15+08:00" level=info msg=" a lot of apples on the tree" fruit=apple size=20
```

### 6. fields

在使用 logrus 时，鼓励用 log.WithFields(log.Fields{}).Fatal() 这种方式替代 og.Fatalf("Failed to send event %s to topic %s with key %d")， 也就是不是用 %s，%d 这种方式格式化，而是直接传入变量 event，topic 给 log.Fields ，这样就显得结构化日志输出，很人性化美观。

```shell
log.WithFields(log.Fields{
  "event": event,
  "topic": topic,
  "key": key,
}).Fatal("Failed to send event")
```

### 7. 设置默认字段

比如在链路追踪里，会有一个 rquest_id ，trace_id 等，想这个 log 一直带有这 2 个字段，logrus 怎么设置？

可以用 log.WithFields(log.Fields{"request_id": request_id, "trace_id": trace_id})

```go
requestLogger := log.WithFields(log.Fields{"request_id": request_id, "trace_id": trace_id})
requestLogger.Info("something happened on that request")
requestLogger.Warn("something not great happened")
```
例子：
```go
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
```

### 8. hook 钩子-扩展logrus功能

hook 给 logrus 提供了强大的可扩展功能.

用户可以给 logrus 编写钩子插件，根据自己的日志需求编写 hook。

logrus 也有一些内置插件[hooks](https://github.com/sirupsen/logrus/blob/master/hooks)。

第三方给 logrus 编写的 hook， [第三方hook列表](https://github.com/sirupsen/logrus/wiki/Hooks)。

官方的 [syslog hook example](https://github.com/sirupsen/logrus/blob/master/hooks/syslog/README.md)：

```go
package main

import (
	"log/syslog"

	"github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"
)

func main() {
	log := logrus.New()
	hook, err := lSyslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
	if err != nil {
		log.Hooks.Add(hook)
	}
}
```

----

也可以到我的公众号 [九卷沉思录-golang常用库包：日志记录库-logrus使用](https://mp.weixin.qq.com/s/ZDwCJq7n9omY1M4lIv4GXQ) 讨论



## 参考

- https://github.com/sirupsen/logrus
- https://github.com/natefinch/lumberjack