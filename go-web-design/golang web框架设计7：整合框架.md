**golang web framework 框架系列文章：**

- [7. golang web框架设计7：整合框架](https://www.cnblogs.com/jiujuan/p/11899010.html)
- [6. golang web框架设计6：上下文设计](https://www.cnblogs.com/jiujuan/p/11898983.html)
- [5. golang web框架设计5：配置设计](https://www.cnblogs.com/jiujuan/p/11898928.html)
- [4. golang web框架设计4：日志设计](https://www.cnblogs.com/jiujuan/p/11898825.html)
- [3. golang web框架设计3：controller设计](https://www.cnblogs.com/jiujuan/p/11898798.html)
- [2. golang web框架设计2：自定义路由](https://www.cnblogs.com/jiujuan/p/11898745.html)
- [1. golang web框架设计1：框架规划](https://www.cnblogs.com/jiujuan/p/11898714.html)

把前面写好的路由器，控制器，日志，都整合在一起

## 全局变量和初始化
定义一些框架的全局变量
```
var (
    BeeApp *App
    AppName string
    AppPath string
    StaticDir map[string]string
    HttpAddr string
    HttpPort int
    RecoverPanic bool
    AutoRender bool
    ViewsPath string
    RunMode string
    AppConfig *Config
)
```

配置文件初始化：
```
func init() {
    BeeApp = NewApp()
    AppPath, _ = os.Getwd()
    StaticDir = make(map[string]string)
    var err error
    AppConfig, err = LoadConfig(path.Join(AppPath, "conf", "app.conf"))
    if err != nil {
        Trace("open config err: ", err)
        HttpAddr = ""
        HttpPort = 8080
        AppName = "beego"
        RunMode = "prod"
        AutoRender = true
        RecoverPanic = true
        ViewsPath = "views"
    } else {
        HttpAddr = AppConfig.String("httpaddr")
        if v, err := AppConfig.Int("httpport"); err != nil {
            HttpPort = 8080
        } else {
            HttpPort = v
        }
        AppName = AppConfig.String("appname")
        if runmode := AppConfig.String("runmode"); runmode != "" {
            RunMode = runmode
        } else {
            RunMode = "prod"
        }
        if ar, err := AppConfig.Bool("autorender"); err != nil {
            AutoRender = true
        } else {
            AutoRender = ar
        }
        if ar, err := AppConfig.Bool("autorecover"); err != nil {
            RecoverPanic = true
        } else {
            RecoverPanic = ar
        }
        if views := AppConfig.String("viewspath"); views == "" {
            ViewsPath = "views"
        } else {
            ViewsPath = views
        }
    }
    StaticDir["/static"] = "static"
}
```

## 完整代码
> [代码地址 beego.go](https://github.com/jiujuan/beego/blob/master/beego.go)

## 简单使用
```
package main

import (
    "github.com/jiujuan/beego"
)

type MainController struct {
    beego.Controller
}

func (c *MainController) Get() {
    c.Ctx.WriteString("hello world")
}

func main() {
    beego.BeeApp.RegisterController("/", &MainController{})
    beego.BeeApp.Run()
}
```