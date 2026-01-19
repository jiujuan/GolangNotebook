## 简介
golang 里的 http 标准库，发起 http 请求时，写法比较繁琐。所以智慧又“偷懒的”程序员们，发挥自己的创造力，写出了一些好用的第三方库，这里介绍其中的一个 http 库：[go-resty](https://github.com/go-resty/resty)

## go-resty 特性
[go-resty](https://github.com/go-resty/resty) 有很多特性：
- 发起 GET, POST, PUT, DELETE, HEAD, PATCH, OPTIONS, etc. 请求
- 简单的链式书写
- 自动解析 JSON 和 XML 类型的文档
- 上传文件
- 重试功能
- 客户端测试功能
- Resty client
- Custom Root Certificates and Client Certificates
- ... ....
等等很多特性。

go-resty更多功能特性请查看文档：[go-resty features](https://github.com/go-resty/resty#features)

## go-resty 使用
> go-resty: v2.3.0

> 用法 example：[https://github.com/go-resty/resty#usage](https://github.com/go-resty/resty#usage)

### 简单的GET
simple_get.go
```go
package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func main() {
	client := resty.New() // 创建一个restry客户端
	resp, err := client.R().EnableTrace().Get("https://httpbin.org/get")

	// Explore response object
	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()

	// Explore trace info
	fmt.Println("Request Trace Info:")
	ti := resp.Request.TraceInfo()
	fmt.Println("  DNSLookup     :", ti.DNSLookup)
	fmt.Println("  ConnTime      :", ti.ConnTime)
	fmt.Println("  TCPConnTime   :", ti.TCPConnTime)
	fmt.Println("  TLSHandshake  :", ti.TLSHandshake)
	fmt.Println("  ServerTime    :", ti.ServerTime)
	fmt.Println("  ResponseTime  :", ti.ResponseTime)
	fmt.Println("  TotalTime     :", ti.TotalTime)
	fmt.Println("  IsConnReused  :", ti.IsConnReused)
	fmt.Println("  IsConnWasIdle :", ti.IsConnWasIdle)
	fmt.Println("  ConnIdleTime  :", ti.ConnIdleTime)
	// fmt.Println("  RequestAttempt:", ti.RequestAttempt)
	// fmt.Println("  RemoteAddr    :", ti.RemoteAddr.String())
}
```

### 增强的GET
```go
client := resty.New()
	resp, err := client.R().
		SetQueryParams(map[string]string{
			"page_no": "1",
			"limit":   "20",
			"sort":    "name",
			"order":   "asc",
			"random":  strconv.FormatInt(time.Now().Unix(), 10),
		}).
		SetHeader("Accept", "application/json").
		SetAuthToken("BC594900518B4F7EAC75BD37F019E08FBC594900518B4F7EAC75BD37F019E08F").
		Get("/search_result")

	// Request.SetQueryString method
	resp, err := client.R().
		SetQueryString("productId=232&template=fresh-sample&cat=resty&source=google&kw=buy a lot more").
		SetHeader("Accept", "application/json").
		SetAuthToken("BC594900518B4F7EAC75BD37F019E08FBC594900518B4F7EAC75BD37F019E08F").
		Get("/show_product")

	// 解析返回的内容，内容是json解析到struct
	resp, err := client.R().
		SetResult(result).
		ForceContentType("application/json").
		Get("v2/alpine/mainfestes/latest")
```

### 各种POST方法
>doc:
>https://github.com/go-resty/resty#various-post-method-combinations

```go

// Create a Resty Clientclient := resty.New()

// POST JSON string// No need to set content type, if you have client level settingresp, err := client.R().
      SetHeader("Content-Type", "application/json").
      SetBody(`{"username":"testuser", "password":"testpass"}`).
      SetResult(&AuthSuccess{}).    // or SetResult(AuthSuccess{}).
      Post("https://myapp.com/login")

// POST []byte array// No need to set content type, if you have client level settingresp, err := client.R().
      SetHeader("Content-Type", "application/json").
      SetBody([]byte(`{"username":"testuser", "password":"testpass"}`)).
      SetResult(&AuthSuccess{}).    // or SetResult(AuthSuccess{}).
      Post("https://myapp.com/login")

// POST Struct, default is JSON content type. No need to set oneresp, err := client.R().
      SetBody(User{Username: "testuser", Password: "testpass"}).
      SetResult(&AuthSuccess{}).    // or SetResult(AuthSuccess{}).
      SetError(&AuthError{}).       // or SetError(AuthError{}).
      Post("https://myapp.com/login")

// POST Map, default is JSON content type. No need to set oneresp, err := client.R().
      SetBody(map[string]interface{}{"username": "testuser", "password": "testpass"}).
      SetResult(&AuthSuccess{}).    // or SetResult(AuthSuccess{}).
      SetError(&AuthError{}).       // or SetError(AuthError{}).
      Post("https://myapp.com/login")

// POST of raw bytes for file upload. For example: upload file to DropboxfileBytes, _ := ioutil.ReadFile("/Users/jeeva/mydocument.pdf")

// See we are not setting content-type header, since go-resty automatically detects Content-Type for youresp, err := client.R().
      SetBody(fileBytes).
      SetContentLength(true).          // Dropbox expects this value
      SetAuthToken("<your-auth-token>").
      SetError(&DropboxError{}).       // or SetError(DropboxError{}).
      Post("https://content.dropboxapi.com/1/files_put/auto/resty/mydocument.pdf") // for upload Dropbox supports PUT too

// Note: resty detects Content-Type for request body/payload if content type header is not set.//   * For struct and map data type defaults to 'application/json'//   * Fallback is plain text content type
```

### PUT
```go

// Note: This is one sample of PUT method usage, refer POST for more combination

// Create a Resty Clientclient := resty.New()

// Request goes as JSON content type// No need to set auth token, error, if you have client level settingsresp, err := client.R().
      SetBody(Article{
        Title: "go-resty",
        Content: "This is my article content, oh ya!",
        Author: "Jeevanandam M",
        Tags: []string{"article", "sample", "resty"},
      }).
      SetAuthToken("C6A79608-782F-4ED0-A11D-BD82FAD829CD").
      SetError(&Error{}).       // or SetError(Error{}).
      Put("https://myapp.com/article/1234")
```

### PATCH
```go

// Note: This is one sample of PUT method usage, refer POST for more combination

// Create a Resty Clientclient := resty.New()

// Request goes as JSON content type// No need to set auth token, error, if you have client level settingsresp, err := client.R().
      SetBody(Article{
        Tags: []string{"new tag1", "new tag2"},
      }).
      SetAuthToken("C6A79608-782F-4ED0-A11D-BD82FAD829CD").
      SetError(&Error{}).       // or SetError(Error{}).
      Patch("https://myapp.com/articles/1234")
```

### DELETE, HEAD, OPTIONS
```go

// Create a Resty Clientclient := resty.New()

// DELETE a article// No need to set auth token, error, if you have client level settingsresp, err := client.R().
      SetAuthToken("C6A79608-782F-4ED0-A11D-BD82FAD829CD").
      SetError(&Error{}).       // or SetError(Error{}).
      Delete("https://myapp.com/articles/1234")

// DELETE a articles with payload/body as a JSON string// No need to set auth token, error, if you have client level settingsresp, err := client.R().
      SetAuthToken("C6A79608-782F-4ED0-A11D-BD82FAD829CD").
      SetError(&Error{}).       // or SetError(Error{}).
      SetHeader("Content-Type", "application/json").
      SetBody(`{article_ids: [1002, 1006, 1007, 87683, 45432] }`).
      Delete("https://myapp.com/articles")

// HEAD of resource// No need to set auth token, if you have client level settingsresp, err := client.R().
      SetAuthToken("C6A79608-782F-4ED0-A11D-BD82FAD829CD").
      Head("https://myapp.com/videos/hi-res-video")

// OPTIONS of resource// No need to set auth token, if you have client level settingsresp, err := client.R().
      SetAuthToken("C6A79608-782F-4ED0-A11D-BD82FAD829CD").
      Options("https://myapp.com/servers/nyc-dc-01")
```

### Requst and Response Middleware
```go

// Create a Resty Clientclient := resty.New()

// Registering Request Middlewareclient.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
    // Now you have access to Client and current Request object
    // manipulate it as per your need

    return nil  // if its success otherwise return error
  })

// Registering Response Middlewareclient.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
    // Now you have access to Client and current Response object
    // manipulate it as per your need

    return nil  // if its success otherwise return error
  })
```

### 重试 Retries
```go

// Create a Resty Clientclient := resty.New()

// Retries are configured per clientclient.
    // Set retry count to non zero to enable retries
    SetRetryCount(3).
    // You can override initial retry wait time.
    // Default is 100 milliseconds.
    SetRetryWaitTime(5 * time.Second).
    // MaxWaitTime can be overridden as well.
    // Default is 2 seconds.
    SetRetryMaxWaitTime(20 * time.Second).
    // SetRetryAfter sets callback to calculate wait time between retries.
    // Default (nil) implies exponential backoff with jitter
    SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
        return 0, errors.New("quota exceeded")
    })
```
Above setup will result in resty retrying requests returned non nil error up to 3 times with delay increased after each attempt.

You can optionally provide client with custom retry conditions:
```go

// Create a Resty Clientclient := resty.New()

client.AddRetryCondition(
    // RetryConditionFunc type is for retry condition function
    // input: non-nil Response OR request execution error
    func(r *resty.Response) (bool, error) {
        return r.StatusCode() == http.StatusTooManyRequests
    },
)
```

### 多个客户端请求
```go

// Here you go!// Client 1client1 := resty.New()
client1.R().Get("http://httpbin.org")
// ...

// Client 2client2 := resty.New()
client2.R().Head("http://httpbin.org")
// ...

// Bend it as per your need!!!
```
---------
也可以到我的公众号：[九卷沉思录-go-resty 库使用详解](https://mp.weixin.qq.com/s?__biz=MjM5NTcyOTY4Mg==&mid=2247484290&idx=1&sn=9492d8aac4ff78678d1ffcc24377789f&chksm=a6f55bff9182d2e9e074d2c75fcfdaf52ccd907c39bc1febc574adf5dfee6985a86395139dad#rd) 讨论

## 参考
- [https://github.com/go-resty/resty#resty](https://github.com/go-resty/resty#resty)