[golang常用库：gorilla/mux-http路由库使用](https://www.cnblogs.com/jiujuan/p/12768907.html)
[golang常用库：配置文件解析库/管理工具-viper使用](https://www.cnblogs.com/jiujuan/p/13799976.html)
[golang常用库：操作数据库的orm框架-gorm基本使用](https://www.cnblogs.com/jiujuan/p/12676195.html)
[golang常用库：字段参数验证库-validator使用](https://www.cnblogs.com/jiujuan/p/13823864.html)

## 一、viper简介

[viper](https://github.com/spf13/viper) 配置管理解析库，是由大神 [Steve Francia](https://github.com/spf13) 开发，他在google领导着 [golang](https://github.com/golang) 的产品开发，他也是 [gohugo.io](https://github.com/gohugoio) 的创始人之一，命令行解析库 [cobra](https://github.com/spf13/cobra) 开发者。总之，他在golang领域是专家，很牛的一个人。

他的github地址：[https://github.com/spf13](https://github.com/spf13)

viper是一个配置管理的解决方案，它能够从 json，toml，ini，yaml，hcl，env 等多种格式文件中，读取配置内容，它还能从一些远程配置中心读取配置文件，如consul，etcd等；它还能够监听文件的内容变化。

viper的 logo：
![](https://img2020.cnblogs.com/blog/650581/202010/650581-20201011224724478-1312624445.png)

## 二、viper功能介绍

- 读取 json，toml，ini，yaml，hcl，env 等格式的文件内容
- 读取远程配置文件，如 consul，etcd 等和监控配置文件变化
- 读取命令行 flag 的值
- 从 buffer 中读取值

配置文件又可以分为不同的环境，比如dev，test，prod等。

[viper](https://github.com/spf13/viper) 可以帮助你专注配置文件管理。

[viper](https://github.com/spf13/viper) 读取配置文件的优先顺序，从高到低，如下：
- 显式设置的Set函数
- 命令行参数
- 环境变量
- 配置文件
- 远程k-v 存储系统，如consul，etcd等
- 默认值

> Viper 配置key是不区分大小写的。

其实，上面的每一种文件格式，都有一些比较有名的解析库，如：
- toml ：[https://github.com/BurntSushi/toml](https://github.com/BurntSushi/toml)
- json ：json的解析库比较多，下面列出几个常用的
  - [https://github.com/json-iterator/go](github.com/json-iterator/go)
  - [https://github.com/mailru/easyjson](https://github.com/mailru/easyjson)
  - [https://github.com/bitly/go-simplejson](https://github.com/bitly/go-simplejson)
  - [https://github.com/tidwall/gjson](https://github.com/tidwall/gjson)
- ini : [https://github.com/go-ini/ini](https://github.com/go-ini/ini)
等等单独文件格式解析库。

但是为啥子要用viper，因为它是一个综合文件解析库，包含了上面所有的文件格式解析，是一个集合体，少了配置多个库的烦恼。

## 三、viper使用

安装viper命令：
`go get github.com/spf13/viper`

>文档: https://github.com/spf13/viper/blob/master/README.md#putting-values-into-viper

### 通过viper.Set设置值
如果某个键通过viper.Set设置了值，那么这个值读取的优先级最高
```go
viper.Set("mysql.info", "this is mysql info")
```

### 设置默认值
>https://github.com/spf13/viper/blob/master/README.md#establishing-defaults

viper 支持默认值的设置。如果配置文件、环境变量、远程配置中没有设置键值，就可以通过viper设置一些默认值。

Examples：
```go
viper.SetDefault("ContentDir", "content")
viper.SetDefault("LayoutDir", "layouts")
viper.SetDefault("Taxonomies", map[string]string{"tag": "tags", "category": "categories"})
```

### 读取配置文件
>https://github.com/spf13/viper/blob/master/README.md#reading-config-files

#### 读取配置文件说明

**读取配置文件要求**：最少要知道从哪个位置查找配置文件。用户一定要设置这个路径。

viper可以从多个路径搜索配置文件，单个viper实例只支持单个配置文件。
viper本身没有设置默认的搜索路径，需要用户自己设置默认路径。

**viper搜索和读取配置文件例子片段**：
```go
viper.SetConfigName("config") // 配置文件的文件名，没有扩展名，如 .yaml, .toml 这样的扩展名
viper.SetConfigType("yaml")  // 设置扩展名。在这里设置文件的扩展名。另外，如果配置文件的名称没有扩展名，则需要配置这个选项
viper.AddConfigPath("/etc/appname/") // 查找配置文件所在路径
viper.AddConfigPath("$HOME/.appname") // 多次调用AddConfigPath，可以添加多个搜索路径
viper.AddConfigPath(".")             // 还可以在工作目录中搜索配置文件
err := viper.ReadInConfig()       // 搜索并读取配置文件
if err != nil { // 处理错误
  panic(fmt.Errorf("Fatal error config file: %s \n", err))
}
```
>说明：
>这里执行viper.ReadInConfig()之后，viper才能确定到底用哪个文件，viper按照上面的AddConfigPath() 进行搜索，找到第一个名为 config.**ext** (**这里的ext代表扩展名**： 如 json,toml,yaml,yml,ini,prop 等扩展名) 的文件后即停止搜索。

>如果有多个名称为config的配置文件，viper怎么搜索呢？它会按照如下顺序搜索
>- config.json
>- config.toml
>- config.yaml
>- config.yml
>- config.properties (这种一般是java中的配置文件名)
>- config.props (这种一般是java中的配置文件名)

你还可以处理一些特殊情况：
```go
if err := viper.ReadInConfig(); err != nil {
    if _, ok := err.(viper.ConfigFileNotFoundError); ok {        
        // 配置文件没有找到; 如果需要可以忽略
    } else {        
        // 查找到了配置文件但是产生了其它的错误
    }
}

// 查找到配置文件并解析成功
```
> **注意[自1.6起]**：  你也可以有不带扩展名的文件，并以编程方式指定其格式。对于位于用户$HOME目录中的配置文件没有任何扩展名，如.bashrc。


### 例子1. 读取配置文件

config.toml 配置文件：
```toml
# this is a toml 

title = "toml exaples"
redis = "127.0.0.1:3300"  # redis

[mysql]
host = "192.168.1.1"
ports = 3306
username = "root"
password = "root123456"
```

viper_toml.go:
```go
package main

import(
    "fmt"
    "github.com/spf13/viper"
)

// 读取配置文件config
type Config struct {
    Redis string
    MySQL MySQLConfig
}

type MySQLConfig struct {
    Port int
    Host string
    Username string
    Password string
}

func main() {
    // 把配置文件读取到结构体上
    var config Config
    
    viper.SetConfigName("config")
    viper.AddConfigPath(".")
    err := viper.ReadInConfig()
    if err != nil {
        fmt.Println(err)
        return
    }
     
    viper.Unmarshal(&config) //将配置文件绑定到config上
    fmt.Println("config: ", config, "redis: ", config.Redis)
}
```

### 例子2. 读取多个配置文件

在例子1基础上多增加一个json的配置文件，config3.json 配置文件：

```json
{
  "redis": "127.0.0.1:33000",
  "mysql": {
    "port": 3306,
    "host": "127.0.0.1",
    "username": "root",
    "password": "123456"
  }
}
```
viper_multi.go
```go
package main

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Redis string
	MySQL MySQLConfig
}

type MySQLConfig struct {
	Port     int
	Host     string
	Username string
	Password string
}

func main() {
	// 读取 toml 配置文件
	var config1 Config

	vtoml := viper.New()
	vtoml.SetConfigName("config")
	vtoml.SetConfigType("toml")
	vtoml.AddConfigPath(".")

	if err := vtoml.ReadInConfig(); err != nil {
		fmt.Println(err)
		return
	}

	vtoml.Unmarshal(&config1)
	fmt.Println("read config.toml")
	fmt.Println("config: ", config1, "redis: ", config1.Redis)

	// 读取 json 配置文件
	var config2 Config
	vjson := viper.New()
	vjson.SetConfigName("config3")
	vjson.SetConfigType("json")
	vjson.AddConfigPath(".")

	if err := vjson.ReadInConfig(); err != nil {
		fmt.Println(err)
		return
	}

	vjson.Unmarshal(&config2)
	fmt.Println("read config3.json")
	fmt.Println("config: ", config1, "redis: ", config1.Redis)
}
```
运行：
>$ go run viper_multi.go
>
>read config.toml
>config:  {127.0.0.1:33000 {0 192.168.1.1 root 123456}} redis:  127.0.0.1:33000
>read config3.json
>config:  {127.0.0.1:33000 {0 192.168.1.1 root 123456}} redis:  127.0.0.1:33000

### 例子3. 读取配置项的值

新建文件夹 item， 在里面创建文件 config.json，内容如下：

```json
{
  "redis": "127.0.0.1:33000",
  "mysql": {
    "port": 3306,
    "host": "127.0.0.1",
    "username": "root",
    "password": "123456",
    "ports": [
        5799,
        6029
    ],
    "metric": {
        "host": "127.0.0.1",
        "port": 2112
    }
  }
}
```

item/viper_get_item.go 读取配置项的值   

```go
package main

import (
	"fmt"

	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() //根据上面配置加载文件
	if err != nil {
		fmt.Println(err)
		return
	}

	host := viper.Get("mysql.host")
	username := viper.GetString("mysql.username")
	port := viper.GetInt("mysql.port")
	portsSlice := viper.GetIntSlice("mysql.ports")

	metricPort := viper.GetInt("mysql.metric.port")
	redis := viper.Get("redis")

	mysqlMap := viper.GetStringMapString("mysql")

	if viper.IsSet("mysql.host") {
		fmt.Println("[IsSet()]mysql.host is set")
	} else {
		fmt.Println("[IsSet()]mysql.host is not set")
	}
	fmt.Println("mysql - host: ", host, ", username: ", username, ", port: ", port)
	fmt.Println("mysql ports :", portsSlice)
	fmt.Println("metric port: ", metricPort)
	fmt.Println("redis - ", redis)

	fmt.Println("mysqlmap - ", mysqlMap, ", username: ", mysqlMap["username"])
}
```

运行：
>$ go run viper_get_item.go
>
>[IsSet()]mysql.host is set
>mysql - host:  127.0.0.1 , username:  root , port:  3306
>mysql ports : [5799 6029]
>metric port:  2112
>redis -  127.0.0.1:33000
>mysqlmap -  map[host:127.0.0.1 metric: password:123456 port:3306 ports: username:root] , username:  root

如果把上面的文件config.json写成toml格式，怎么解析？ 改成config1.toml:

```toml
# toml
toml = "toml example"

redis = "127.0.0.1:33000"

[mysql]
port = 3306
host = "127.0.0.1"
username = "root"
password = "123456"
ports = [5799,6029]
[mysql.metric]
host = "127.0.0.1"
port = 2112
```

其实解析代码差不多，只需修改2处，
>viper.SetConfigName("config") 里的 config 改成 config1 ，
>viper.SetConfigType("json")里的 json 改成 toml，其余代码都一样。解析的效果也一样。

**viper获取值的方法：**
- Get(key string) : interface{}
- GetBool(key string) : bool
- GetFloat64(key string) : float64
- GetInt(key string) : int
- GetIntSlice(key string) : []int
- GetString(key string) : string
- GetStringMap(key string) : map[string]interface{}
- GetStringMapString(key string) : map[string]string
- GetStringSlice(key string) : []string
- GetTime(key string) : time.Time
- GetDuration(key string) : time.Duration
- IsSet(key string) : bool
- AllSettings() : map[string]interface{}

### 例子4. 读取命令行的值

新建文件夹 cmd，然后cmd文件夹里新建config.json文件：
```json
{
  "redis":{
    "port": 3301,
    "host": "127.0.0.1"
  },
  "mysql": {
    "port": 3306,
    "host": "127.0.0.1",
    "username": "root",
    "password": "123456"
  }
}
```
go解析文件，cmd/viper_pflag.go：
```go
package main

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	pflag.Int("redis.port", 3302, "redis port")

	viper.BindPFlags(pflag.CommandLine)
	pflag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() //根据上面配置加载文件
	if err != nil {
		fmt.Println(err)
		return
	}

	host := viper.Get("mysql.host")
	username := viper.GetString("mysql.username")
	port := viper.GetInt("mysql.port")
	redisHost := viper.GetString("redis.host")
	redisPort := viper.GetInt("redis.port")

	fmt.Println("mysql - host: ", host, ", username: ", username, ", port: ", port)
	fmt.Println("redis - host: ", redisHost, ", port: ", redisPort)
}
```

**1.不加命令行参数运行：**

>$ go run viper_pflag.go
>
>mysql - host:  127.0.0.1 , username:  root , port:  3306
>redis - host:  127.0.0.1 , port:  3301

说明：redis.port 的值是 3301，是 config.json 配置文件里的值。

**2.加命令行参数运行**
>$ go run viper_pflag.go --redis.port 6666
>
>mysql - host:  127.0.0.1 , username:  root , port:  3306
>redis - host:  127.0.0.1 , port:  6666

说明：加了命令行参数 `--redis.port 6666`，这时候redis.port输出的值为 `6666`，读取的是cmd命令行的值

### 例子5：io.Reader中读取值

>https://github.com/spf13/viper#reading-config-from-ioreader

viper_ioreader.go
```go
package main

import (
	"bytes"
	"fmt"

	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigType("yaml")

	var yaml = []byte(`
Hacker: true
name: steve
hobbies:
- skateboarding
- snowboarding
- go
clothing:
  jacket: leather
  trousers: denim
age: 35
eyes : brown
beard: true
    `)

	err := viper.ReadConfig(bytes.NewBuffer(yaml))
	if err != nil {
		fmt.Println(err)
		return
	}
	hacker := viper.GetBool("Hacker")
	hobbies := viper.GetStringSlice("hobbies")
	jacket := viper.Get("clothing.jacket")
	age := viper.GetInt("age")
	fmt.Println("Hacker: ", hacker, ",hobbies: ", hobbies, ",jacket: ", jacket, ",age: ", age)

}
```

### 例子6：写配置文件

>https://github.com/spf13/viper#writing-config-files

新建文件 writer/viper_write_config.go:
```go
package main

import (
	"fmt"

	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.Set("yaml", "this is a example of yaml")

	viper.Set("redis.port", 4405)
	viper.Set("redis.host", "127.0.0.1")

	viper.Set("mysql.port", 3306)
	viper.Set("mysql.host", "192.168.1.0")
	viper.Set("mysql.username", "root123")
	viper.Set("mysql.password", "root123")

	if err := viper.WriteConfig(); err != nil {
		fmt.Println(err)
	}
}
```
运行：
>$ go run viper_write_config.go

没有任何输出表示生成配置文件成功
```yaml
mysql:
  host: 192.168.1.0
  password: root123
  port: 3306
  username: root123
redis:
  host: 127.0.0.1
  port: 4405
yaml: this is a example of yaml
```
#### WriteConfig() 和 SafeWriteConfig() 区别:
>如果待生成的文件已经存在，那么SafeWriteConfig()就会报错，`Config File "config.yaml" Already Exists`， 而WriteConfig()则会直接覆盖同名文件。

---

> 也欢迎到我的公众号：[九卷沉思录](https://mp.weixin.qq.com/s/rq02tQTNxUvgG0i4DzsiYQ) 继续讨论该文章

## 四、参考

- [viper 文档](https://github.com/spf13/viper/blob/master/README.md)
- [golang json库gjson的使用](https://www.jianshu.com/p/623f8ca5ec12)

