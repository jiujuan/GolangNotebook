> 继续学习golang web框架设计

**golang web framework 框架系列文章：**

- [7. golang web框架设计7：整合框架](https://www.cnblogs.com/jiujuan/p/11899010.html)

- [6. golang web框架设计6：上下文设计](https://www.cnblogs.com/jiujuan/p/11898983.html)

- [5. golang web框架设计5：配置设计](https://www.cnblogs.com/jiujuan/p/11898928.html)

- [4. golang web框架设计4：日志设计](https://www.cnblogs.com/jiujuan/p/11898825.html)

- [3. golang web框架设计3：controller设计](https://www.cnblogs.com/jiujuan/p/11898798.html)

- [2. golang web框架设计2：自定义路由](https://www.cnblogs.com/jiujuan/p/11898745.html)

- [1. golang web框架设计1：框架规划](https://www.cnblogs.com/jiujuan/p/11898714.html)

  
## controller作用

MVC设计模式里面的这个C，控制器。
- Model是后台返回的数据；
- View是渲染页面，通常是HTML的模板页面；
- Controller是处理不同URL的控制器

Controller在整个MVC框架中起到一个核心的纽带作用，负责处理业务逻辑，因此控制器是整个框架必不可少的部分，Model和View有时候可以没有，例如没有数据处理的业务逻辑，没有页面的302等


## Controller设计

前面小结路由实现注册了struct的功能，而struct中实现了REST方式，所以我们要设计一个逻辑处理controller的基类，设计2个类型，一个struct，一个interface
```
type Controller struct {
    Ct *Context
    Tpl *template.Template
    Data map[interface{}]interface{}
    ChildName string
    TplNames  string
    Layout    []string
    TplExt    string
}
```

```
type ControllerInterface interface {
    Init(ct *Context, cn string)
    Prepare()
    Get()
    Post()
    Delete()
    Put()
    Head()
    Patch()
    Options()
    Finish()
    Render() error
}
```

在前面第2节`自定义路由`中的add函数，第二个参数ControllerInterface 就是这里定义的，因此，只要我们实现这个接口就可以了，所以我的基类Controller实现如下方法：

```
func (c *Controller) Init(ct *Context, cn string) {
    c.Data = make(map[interface{}]interface{})
    c.Layout = make([]string, 0)
    c.TplNames = ""
    c.ChildName = cn
    c.Ct = ct
    c.TplExt = "tpl"
}

func (c *Controller) Prepare() {
    
}

func (c *Controller) Finish() {
    
}

func (c *Controller) Get() {
    http.Error(c.Ct.ResponseWriter, "Method Not Allowed", 405)
}

```

## 完整代码：

> [代码地址 controller.go](https://github.com/jiujuan/beego/blob/master/controller.go)