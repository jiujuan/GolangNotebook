package main

import (
	"fmt"
	"sync"
)

// Result 用于存储各个任务的执行结果
type Result struct {
	Source string
	Data   string
}

func main() {
	var wg sync.WaitGroup
	results := make(chan Result, 3) // 使用缓冲通道存储结果

	// 数据源列表
	sources := []string{"用户数据库", "Redis缓存", "外部API"}

	// 为每个数据源启动一个 goroutine
	for _, source := range sources {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			// 模拟从不同数据源获取数据
			data := fetchFromSource(src)
			results <- Result{Source: src, Data: data}
		}(source)
	}

	// 启动一个 goroutine 等待所有任务完成后关闭通道
	go func() {
		wg.Wait()
		close(results)
	}()

	// 主 goroutine 收集并处理结果
	fmt.Println("开始并行获取数据...")
	allResults := make([]Result, 0, len(sources))
	for result := range results {
		fmt.Printf("收到来自 %s 的数据: %s\n", result.Source, result.Data)
		allResults = append(allResults, result)
	}

	fmt.Printf("\n共获取到 %d 条数据，准备进行后续处理\n", len(allResults))
}

// fetchFromSource 模拟从不同数据源获取数据的函数
func fetchFromSource(source string) string {
	// 这里只是模拟
	return fmt.Sprintf("来自%s的数据内容", source)
}