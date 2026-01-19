**golang web framework 框架系列文章：**
- [7. golang web框架设计7：整合框架](https://www.cnblogs.com/jiujuan/p/11899010.html)
- [6. golang web框架设计6：上下文设计](https://www.cnblogs.com/jiujuan/p/11898983.html)
- [5. golang web框架设计5：配置设计](https://www.cnblogs.com/jiujuan/p/11898928.html)
- [4. golang web框架设计4：日志设计](https://www.cnblogs.com/jiujuan/p/11898825.html)
- [3. golang web框架设计3：controller设计](https://www.cnblogs.com/jiujuan/p/11898798.html)
- [2. golang web框架设计2：自定义路由](https://www.cnblogs.com/jiujuan/p/11898745.html)
- [1. golang web框架设计1：框架规划](https://www.cnblogs.com/jiujuan/p/11898714.html)

context，翻译为上下文，为什么要设计这个结构？就是把http的请求和响应，以及参数结合在一起，便于集中处理信息，以后框架的扩展等。好多框架比如gin，都是有这个上下文结构。

context结构为
```
type Context struct {
    ResponseWriter http.ResponseWriter
    Request *http.Request
    Params  map[string]string
}
```

操作函数
```
func (ctx *Context) WriteString(content string) {
    ctx.ResponseWriter.Write([]byte(content))
}

func (ctx *Context) Abort(status int, body string) {
    ctx.ResponseWriter.WriteHeader(status)
    ctx.ResponseWriter.Write([]byte(body))
}

func (ctx *Context) Redirect(status int, url string) {
    ctx.ResponseWriter.Header().Set("Location", url)
    ctx.ResponseWriter.WriteHeader(status)
    ctx.ResponseWriter.Write([]byte("Redirecting to: " + url))
}

func (ctx *Context) NotFound(message string) {
    ctx.ResponseWriter.WriteHeader(404)
    ctx.ResponseWriter.Write([]byte(message))
}

func (ctx *Context) ContentType(ext string) {
    if !strings.HasPrefix(ext, ".") {
        ext = "." + ext
    }
    ctype := mime.TypeByExtension(ext)
    if ctype != "" {
        ctx.ResponseWriter.Header().Set("Content-Type", ctype)
    }
}
```

## 完整代码：
> [代码地址 context.go](https://github.com/jiujuan/beego/blob/master/context.go)
>
> 