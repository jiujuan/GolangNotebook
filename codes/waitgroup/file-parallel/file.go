package main

import (
    "bufio"
    "fmt"
    "os"
    "sync"
)

func main() {
    var wg sync.WaitGroup
    files := []string{"data1.txt", "data2.txt", "data3.txt"}

    for _, f := range files {
        // 1. 在启动 goroutine 之前 Add
        wg.Add(1)
        go processFile(f, &wg)
    }

    // 3. 阻塞等待所有任务完成
    wg.Wait()
    fmt.Println("所有文件处理完毕")
}

func processFile(filename string, wg *sync.WaitGroup) {
    // 2. 确保在函数退出时调用 Done
    defer wg.Done()

    file, err := os.Open(filename)
    if err != nil {
        fmt.Printf("Error opening %s: %v\n", filename, err)
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineCount := 0
    for scanner.Scan() {
        lineCount++
    }
    fmt.Printf("File: %s, Lines: %d\n", filename, lineCount)
}