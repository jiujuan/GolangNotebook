**golang web framework 框架系列文章：**
- [7. golang web框架设计7：整合框架](https://www.cnblogs.com/jiujuan/p/11899010.html)
- [6. golang web框架设计6：上下文设计](https://www.cnblogs.com/jiujuan/p/11898983.html)
- [5. golang web框架设计5：配置设计](https://www.cnblogs.com/jiujuan/p/11898928.html)
- [4. golang web框架设计4：日志设计](https://www.cnblogs.com/jiujuan/p/11898825.html)
- [3. golang web框架设计3：controller设计](https://www.cnblogs.com/jiujuan/p/11898798.html)
- [2. golang web框架设计2：自定义路由](https://www.cnblogs.com/jiujuan/p/11898745.html)
- [1. golang web框架设计1：框架规划](https://www.cnblogs.com/jiujuan/p/11898714.html)

配置信息的解析，实现的是一个key=value，键值对的一个配置文件，类似于ini的配置格式，然后解析这个文件，把解析的数据保存到map中，最后调用的时候通过几个string，int之类的函数返回相应的值

首先定义ini配置文件的一些全局性常量：
```
var (
    bComment = []byte{'#'}
    bEmpty = []byte{}
    bEqual = []byte{'='}
    bDQuote = []byte{'"'}
)
```

配置文件的格式：
```
type Config struct {
    filename string
    comment map[int][]string
    data map[string]string
    offset map[string]int64
    sync.RWMutex
}
```

定义解析文件的函数：
解析文件过程是打开文件，然后一行一行读取，解析注释，空行和k=v的数据
```
func LoadConfig(name string) (*Config, error)
```

下面实现一些读取配置文件的函数，返回的值确定为bool，int，int64或string：
```
func (c *Config) Bool(key string) (bool, error) {
    return strconv.ParseBool(c.data[key])
}

func (c *Config) Int(key string) (int, error) {
    return strconv.Atoi(c.data[key])
}

func (c *Config) Float(key string) (float64, error) {
    return strconv.ParseFloat(c.data[key], 64)
}

func (c *Config) String(key string) string {
    return c.data[key]
}
```

## 完整代码：
> [代码地址 config.go](https://github.com/jiujuan/beego/blob/master/config.go)
>
> 