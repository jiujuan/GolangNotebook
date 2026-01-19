先从业务开发角度出发，来逐渐引出中间件。

## 一、刚开始时业务开发

开始业务开发时，业务需求比较少。

1. 当我们最开始进行业务开发时，需求不是很多。 第一个需求产是品向大家打声招呼：“hello world”。

接到需求任务，我们就进行代码开发了。
一般都会写下如下的代码，用handlefunc来处理请求的服务

```Go
package main

import (
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.ListenAndServe(":8080", nil)
}

```

2. 假如现在业务有变化了，我们要新增一个hello服务的处理耗时，怎么做？
这个需求比较简单，修改代码如下：

```Go
package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "", 0)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	timeStart := time.Now()
	w.Write([]byte("hello world"))
	timeElapsed := time.Since(timeStart)
	logger.Println(timeElapsed)
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.ListenAndServe(":8080", nil)
}
```

这样就可以输出当前hello请求到日志文件了。

3. 完成了这个需求后。过了没多久，又有新的需求来了，我们要显示信息，显示Email，
显示好朋友，并且这就一个接口也需要增加耗时记录。

一下子又增加了很多api， 简略示例代码如下：

```Go

package main

func helloHandler(wr http.ResponseWriter, r *http.Request) {
    // ...
}

func showInfoHandler(wr http.ResponseWriter, r *http.Request) {
    // ...
}

func showEmailHandler(wr http.ResponseWriter, r *http.Request) {
    // ...
}

func showFriendsHandler(wr http.ResponseWriter, r *http.Request) {
    timeStart := time.Now()
    wr.Write([]byte("your friends is tom and alex"))
    timeElapsed := time.Since(timeStart)
    logger.Println(timeElapsed)
}

func main() {
    http.HandleFunc("/", helloHandler)
    http.HandleFunc("/info/show", showInfoHandler)
    http.HandleFunc("/email/show", showEmailHandler)
    http.HandleFunc("/friends/show", showFriendsHandler)
    // ...
}
```
每一个handler里面都需要记录运行的时间，每次新加路由都要写同样的代码。都要把业务逻辑代码拷贝过来。

4. 业务继续发展，又有了新的需求，增加一个监控系统，需要你上报接口运行时间到监控系统里面，以便监控接口的稳定性。这个监控系统叫metrics。 

好了，现在你又要修改代码，通过http post方式把耗时时间发送给metrics系统。
而且你要修改好多个handler，增加metrics上报接口代码。

修改代码：

```Go
func helloHandler(wr http.ResponseWriter, r *http.Request) {
    timeStart := time.Now()
    wr.Write([]byte("hello"))
    timeElapsed := time.Since(timeStart)
    logger.Println(timeElapsed)
    // 新增耗时上报
    metrics.Upload("timeHandler", timeElapsed)
}

func showInfoHandler(wr http.ResponseWriter, r *http.Request) {
    // ...
    
    // 新增耗时上报
    metrics.Upload("timeHandler", timeElapsed)
}

func showEmailHandler(wr http.ResponseWriter, r *http.Request) {
    // ...
    
     // 新增耗时上报
    metrics.Upload("timeHandler", timeElapsed)
}

func showFriendsHandler(wr http.ResponseWriter, r *http.Request) {
    timeStart := time.Now()
    wr.Write([]byte("your friends is tom and alex"))
    timeElapsed := time.Since(timeStart)
    logger.Println(timeElapsed)
    
     // 新增耗时上报
    metrics.Upload("timeHandler", timeElapsed)
}
```

到这里，发现要修改好多的handler函数，才能把接口的耗时时间上报到metrics系统里。
随着新需求越来越多，handler也会越多，那么我们修改的地方也就越多。增加了一个简单的业务统计，就要修改好多个handler函数。

虽然只是增加一个业务统计，我们就要去修改handler，来增加这些和业务无关的代码。
一开始我们并没有做错什么， 但是随着业务的发展，我们逐渐陷入了代码的泥潭。

接下来，我们该怎么办呢？怎么处理这种情况？

## 二、业务逐渐多了后

随着业务发展，handler越来越多，增加与业务无关的代码所要修改的地方也越来越多。这时候怎么办？有没有办法可以处理这种情况呢？ 

想一下，java里面有一个Filter的技术，可以拦截请求的处理。我们可不可以利用这个思想来解决我们的问题呢。这种思想当然是可以的。

在go里面就是利用 `http.Handler` 来把你要处理的函数包起来（实际就是拦截了），然后处理。
在go里面有一个学名叫  **middleware**（**中间件**），中间件常见的位置是ServeMux和应用处理程序之间。

http的请求控制流程：

> ServeMux => Middleware Handler => Application Handler

针对上面的需求，我们用中间件的方法来修改下：
```Go
package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "", 0)

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}

func timeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeStart := time.Now()

		//next handler
		next.ServeHTTP(w, r)

		timeElapsed := time.Since(timeStart)
		logger.Println(timeElapsed)
	})
}

func main() {
	http.HandleFunc("/", timeMiddleWare(hello))
	http.ListenAndServe(":8080", nil)
}

```
这样就实现了中间件。

也是把业务代码和非业务代码进行了剥离。

## 三、怎么实现中间件

上面的中间件是怎么实现的呢？

1. 他要满足`http.Handler` 接口
```Go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```
写一个简单的程序：
```Go
func showinfoHandler(info string) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte(info)
  })
}
```
上面程序我们把简单的处理程序w.Write放在了一个匿名函数中，然后引用了外部的info构成了一个闭包。接下来我们用 `http.HandlerFunc` 适配器将此闭包转换为处理程序，然后返回它。

我们可以用相同的方法，将下一个处理程序作为变量来进行传递，然后调用ServeHTTP() 方法将控制转移到下一个处理程序，然后返回它。

```Go
func demoMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // out logic
        
        next.ServeHTTP(w, r)
  })
}
```
上面的中间件函数有一个 `func(http.Handler) http.Handler` 的函数签名。它接收一个处理程序作为参数并返回另外一个处理程序。

#### 一个完整的例子

用一个完整的例子来看看中间件的执行流程：
middlewaredemo.go
```Go
package main

import (
	"log"
	"net/http"
)

func middleOne(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("before middleone")
		next.ServeHTTP(w, r)
		log.Println("after middlewareOne again")
	})
}

func middleTwo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("before middletwo")
		if r.URL.Path == "/foo" {
			return
		}
		next.ServeHTTP(w, r)
		log.Println("after middleTwo again")
	})
}

func final(w http.ResponseWriter, r *http.Request) {
	log.Println("exec finalHandler")
	w.Write([]byte("OK"))
}

func main() {
	finalHandler := http.HandlerFunc(final)

	http.Handle("/", middleOne(middleTwo(finalHandler)))
	http.ListenAndServe(":3030", nil)
}
```
先执行下面的命令
> go run middlewaredemo.go

然后在浏览器上执行：http://localhost:3030/
运行结果：
> 2020/04/19 23:07:14 before middleone
> 2020/04/19 23:07:14 before middletwo
> 2020/04/19 23:07:14 exec finalHandler
> 2020/04/19 23:07:14 after middleTwo again
> 2020/04/19 23:07:14 after middlewareOne again

看看执行结果，在 `next.ServeHTTP`  **前**，看到处理顺序是依次按照嵌套的顺序出结果， 但是在 `next.ServeHTTP` **后** 的程序，是按照相反的方向出结果。

然后在运行 ： http://localhost:3030/foo
运行结果：
>2020/04/19 23:12:15 before middleone
>2020/04/19 23:12:15 before middletwo
>2020/04/19 23:12:15 after middlewareOne again

`middleTwo` 函数里 `return` 后面的程序，都没有显示了。
所以，在中间件中，我们可以用 `return` 来停止在中间件程序的传播。

## 四、Gin框架的中间件

> github.com/gin-gonic/gin，这个web框架使用很广泛，它也有中间件功能。

- 使用方法 一
```Go
// 定义中间件
func middlewareOne(c *gin.Context) {
    // 中间件逻辑
}
```

```Go
// 使用中间件
r := gin.Default()
r.Use(middlewareOne)
```
- 使用方法 二
```Go
func middlewareTwo() gin.HandlerFunc {
    // 自定义逻辑
    return func(c *gin.Context) {
        // 中间件逻辑
    }
}
```

```Go
// 使用中间件
r := gin.Deafult()
r.Use(middlewareTwo()
```

Gin还有一种像java中的Filter，处理前，处理后的一种方法 `Next()`
比如：demo1.go
```Go
package main

import (
	"fmt"
	"time"
	"github.com/gin-gonic/gin"
)

func middlewareOne() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(200, "before middlewareOne: handler "+time.Now().String()+"\n")

		c.Next()

		c.String(200, "after middlewareOne : handler "+time.Now().String()+"\n")
	}
}

func middlewareTwo() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(200, "before middlewareTwo: handler \n")

		c.Next()

		c.String(200, "after middlewareTwo : handler \n")
	}
}

func main() {
	r := gin.Default()

	r.Use(middlewareOne(), middlewareTwo())

	r.GET("/", func(c *gin.Context) {
		fmt.Println("start!")
		c.String(200, "GET method \n")
		fmt.Println("end!")
	})

	r.Run(":8080")
}

```
先在命令行运行： 
>go run demo1.go

然后在浏览器上输入：http://localhost:8080/
就可以看到输出结果：
>before middlewareOne: handler 2020-04-20 01:03:25.4547842 +0800 CST m=+21.318612501
>before middlewareTwo: handler 
>GET method 
>after middlewareTwo : handler 
>after middlewareOne : handler  2020-04-20 01:03:25.4547842 +0800 CST m=+21.318612501

**其实我们看到gin框架实现的中间件，它书写形式并不是像上面的那种嵌套模式。如果有很多中间件的话，那么这种嵌套模式写出来就让人感觉非常复杂。**

**而gin中间件这种书写模式，就很清晰，适合人阅读。是一种优雅的实现方式。**

它是怎么实现的呢？

## 五、Gin中间件的实现

> gin v1.6.2 版本

**gin.go**

```Go
// HandlerFunc defines the handler used by gin middleware as return value.
type HandlerFunc func(*Context)

// HandlersChain defines a HandlerFunc array. 
//定义了Handlers链
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main one.
// 从Handlers里取出最后一个Handler，就是main自己
func (c HandlersChain) Last() HandlerFunc {
    if length := len(c); length > 0 {
        return c[length-1]
    }
    return nil
}

// RouteInfo represents a request route's specification which contains method and path and its handler.
type RouteInfo struct {
    Method      string
    Path        string
    Handler     string
    HandlerFunc HandlerFunc
}
// RoutesInfo defines a RouteInfo array.
type RoutesInfo []RouteInfo

type Engine struct {
    RouterGroup
    // Enables automatic redirection if the current route can't be matched but a
    // handler for the path with (without) the trailing slash exists.
    // For example if /foo/ is requested but a route only exists for /foo, the
    // client is redirected to /foo with http status code 301 for GET requests
    // and 307 for all other request methods.
    RedirectTrailingSlash bool

    ... ...
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
// 初始化Engine，里面就用到了2个中间件函数Logger和Recovery
func Default() *Engine {
    debugPrintWARNINGDefault()
    engine := New()
    engine.Use(Logger(), Recovery())
    return engine
}

// Use attaches a global middleware to the router. ie. the middleware attached though Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
// 增加 middleware -> 实质是到 RouterGroup.Use()
func (engine *Engine) Use(middleware ...HandlerFunc) IRoutes {
    engine.RouterGroup.Use(middleware...)  //到了 RouterGroup 里的Use
    engine.rebuild404Handlers()
    engine.rebuild405Handlers()
    return engine
}
```

**context.go**

```Go

// Context is the most important part of gin. It allows us to pass variables between middleware,
// manage the flow, validate the JSON of a request and render a JSON response for example.
type Context struct {
    writermem responseWriter
    Request   *http.Request
    Writer    ResponseWriter
    Params   Params
    handlers HandlersChain   //这里有一个handlers 链，一个slice
    index    int8
    fullPath string
    engine *Engine
    // This mutex protect Keys map
    KeysMutex *sync.RWMutex
    // Keys is a key/value pair exclusively for the context of each request.
    Keys map[string]interface{}
  
   ... ...
}

// Handler returns the main handler.
// 返回main handler
func (c *Context) Handler() HandlerFunc {
    return c.handlers.Last()
}

// Next should be used only inside middleware.
// It executes the pending handlers in the chain inside the calling handler.
// See example in GitHub.
func (c *Context) Next() {
    c.index++
    for c.index < int8(len(c.handlers)) {
        c.handlers[c.index](c)
        c.index++
    }
}
```

还有一个**routegroup.go**里的handler chain
```Go
// Use adds middleware to the group, see example code in GitHub.
// 增加middleware
func (group *RouterGroup) Use(middleware ...HandlerFunc) IRoutes {
    group.Handlers = append(group.Handlers, middleware...)
    return group.returnObj()
}

func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
    finalSize := len(group.Handlers) + len(handlers)
    if finalSize >= int(abortIndex) {
        panic("too many handlers")
    }
    mergedHandlers := make(HandlersChain, finalSize)
    copy(mergedHandlers, group.Handlers)
    copy(mergedHandlers[len(group.Handlers):], handlers)
    return mergedHandlers
}
```

## 六、改进以前框架

写到了这里，想起了以前写的 [golang web框架](https://www.cnblogs.com/jiujuan/p/11899010.html) 文章，那只是实现了一个简单的MVC功能，并不具备可扩展性，有了这个中间件技术，就可以把以前的框架进行改进。
改进后的[全新 go web 框架 **lilac**](https://github.com/jiujuan/lilac)

先实现功能，然后再进行优化改进 - 论开发。

## 七、参考

- [Go高级语言编程-中间件](https://chai2010.gitbooks.io/advanced-go-programming-book/content/ch5-web/ch5-03-middleware.html) ，这个写的非常非常非常好，业务部分基本是从这里来的
- [gin框架](https://github.com/gin-gonic/gin) 