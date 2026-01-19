**golang web framework 框架系列文章：**

- [7. golang web框架设计7：整合框架](https://www.cnblogs.com/jiujuan/p/11899010.html)
- [6. golang web框架设计6：上下文设计](https://www.cnblogs.com/jiujuan/p/11898983.html)
- [5. golang web框架设计5：配置设计](https://www.cnblogs.com/jiujuan/p/11898928.html)
- [4. golang web框架设计4：日志设计](https://www.cnblogs.com/jiujuan/p/11898825.html)
- [3. golang web框架设计3：controller设计](https://www.cnblogs.com/jiujuan/p/11898798.html)
- [2. golang web框架设计2：自定义路由](https://www.cnblogs.com/jiujuan/p/11898745.html)
- [1. golang web框架设计1：框架规划](https://www.cnblogs.com/jiujuan/p/11898714.html)

beego的日志设计思路来自于seelog，根据不同的level来记录日志，beego设计的日志是一个轻量级的，采用系统log.Logger接口，默认输出到os.Stdout，用户可以实现这个接口然后通过设置beego.SetLogger设置自定义的输出

```
const (
    LevelTrace = iota
    LevelDebug
    LevelInfo
    LevelWarning
    LevelError
    LevelCritical
)

var level = LevelTrace

func Level() int {
    return level
}

func SetLevel(l int) {
    level = l
}
```
上面着这一段实现日志分级，默认级别是Trace，用户可以通过SetLevel可以设置不同的分级

```
func Trace(v ...interface{}) {
    if level <= LevelTrace {
        BeeLogger.Printf("[T] %v\n", v)
    }
}

func Debug(v ...interface{}) {
    if level <= LevelDebug {
        BeeLogger.Printf("[D] %v\n", v)
    }
}
```

## 完整代码：

> [代码地址 log.go](https://github.com/jiujuan/beego/blob/master/log.go)