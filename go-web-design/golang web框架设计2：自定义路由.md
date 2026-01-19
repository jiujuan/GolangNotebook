> 继续学习谢大的Go web框架设计

**golang web framework 框架系列文章：**

- [7. golang web框架设计7：整合框架](https://www.cnblogs.com/jiujuan/p/11899010.html)
- [6. golang web框架设计6：上下文设计](https://www.cnblogs.com/jiujuan/p/11898983.html)
- [5. golang web框架设计5：配置设计](https://www.cnblogs.com/jiujuan/p/11898928.html)
- [4. golang web框架设计4：日志设计](https://www.cnblogs.com/jiujuan/p/11898825.html)
- [3. golang web框架设计3：controller设计](https://www.cnblogs.com/jiujuan/p/11898798.html)
- 
- [2. golang web框架设计2：自定义路由](https://www.cnblogs.com/jiujuan/p/11898745.html)
- [1. golang web框架设计1：框架规划](https://www.cnblogs.com/jiujuan/p/11898714.html)

## HTTP路由

http路由负责将一个http的请求交到对应的函数处理（或者一个struct的方法），路由在框架中相当于一个事件处理器，而这个时间包括
- 用户请求的路径（path）（eg：/user/12, /article/1），当然还有查询信息（eg：?id=12）
- HTTP的请求方法（method）（GET,POST,PUT,DELETE,PATHC等）

## 路由的默认实现

Go的http包设计和实现路由，例子来说明
```
func fooHander(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

http.Handle("/", fooHandler)

http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request){
    fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
})

log.Fatal(http.ListenAndServe(":8080", nil))
```
上面的例子调用的是http默认的DefaultServeMux来添加路由，需要提供2个参数，第一个参数是用户访问资源的URL路径，第二个参数是即将要执行的函数。所以路由思路主要就是2点：
- 1. 添加路由信息
- 2. 更加用户请求转发到执行的函数上

Go默认的路由添加都是通过函数http.Handle和http.HandleFunc 等来添加，底层都是调用了DefaultServeMux.Handle(pattern string, handler Handler)，这个函数会把路由信息存储在一个map信息中map[string]muxEntry， 这个就解决了上面说的第一点。

Go监听端口，然后接收到tcp连接会扔给Handler来处理，上面例子用nil参数，及是用 http.DefaultServeMux，通过DefaultServeMux.ServeHTTP函数进行调度，遍历之前存储的map路由信息，和yoghurt访问的URL进行匹配，用来查询对应的注册函数，这就解决了上面所说的第二点。

## Go默认路由的缺点

- 1.不支持参数设定，例如/user/:uid 这种泛型类型匹配
- 2.无法很友好的支持REST模式，无法限制访问方法，例如上面例子中，用户访问/foo，可以用GET,POST,DELETE,HEAD等方式访问
- 3.一般网站路由规则太多，编写频繁。这种路由较多的可以进一步简化，通过struct方法简化

## 路由设计

针对上面Go默认路由的缺点，首先要解决参数支持就要用到正则， 第二和第三个通过一种变通方法，REST的方法对应到struct的方法中，然后路由到struct而不是函数，这样路由时候就可以根据method来执行不同的方法。

### 存储路由

根据上面说的思路，设计2个数据类型
`controllerInfo`，保存路径和对于的struct，这里是一个reflect.Type类型
一个`ControllerRegistor`，这个是一个slice用来保存用户添加的路由信息
```
type controllerInfo struct {
    regex *regexp.Regexp
    params map[int]string
    controllerType reflect.Type
}
```

```
type ControllerRegistor struct {
    routers []*controllerInfo
}
```

初始化ControllerRegistor
```
func NewControllerRegistor() *ControllerRegistor {
    return &ControllerRegistor{routers: make([]*controllerInfo, 0)}
}
```

ControllerRegistor对外的函数Add，添加url和对于的执行函数
```
func (p *ControllerRegistro) Add(pattern string, c ControllerInterfce)
```
> 在上面的函数中，第二个参数 ControllerInterfce 将在后面一节controller设计中讲解，它是一个interface 类型

还有一个自动路由，实现的是Go定义的函数ServeHTTP

```
func (p *ControllerRegistor) ServeHTTP(rw http.ResponseWriter, r *http.Request) 

```

## 完整代码：
> [代码地址 router.go](https://github.com/jiujuan/beego/blob/master/router.go)