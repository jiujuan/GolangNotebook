package main

import (
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

func fetchURL(url string, wg *sync.WaitGroup, results chan<- string) {
    defer wg.Done()
    
    client := http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        results <- fmt.Sprintf("%s: 错误 - %v", url, err)
        return
    }
    defer resp.Body.Close()
    
    body, _ := io.ReadAll(resp.Body)
    results <- fmt.Sprintf("%s: 状态码 %d, 大小 %d bytes", 
        url, resp.StatusCode, len(body))
}

func main() {
    urls := []string{
        "https://www.baidu.com",
        "https://www.so.com",
    }
    
    var wg sync.WaitGroup
    // 用 channel 来存结果
    results := make(chan string, len(urls))
    
    for _, url := range urls {
        wg.Add(1)
        go fetchURL(url, &wg, results)
    }
    
    // 等待所有请求完成
    wg.Wait()
    close(results)
    
    // 输出结果
    for result := range results {
        fmt.Println(result)
    }
}