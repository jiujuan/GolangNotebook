package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	// 模拟启动 3 个并发任务
	taskCount := 3 
	wg.Add(taskCount)

	for i:=1; i<=taskCount; i++ {
		taskID := i
		go func(){
			defer wg.Done() // 确保函数结束时计数器减一

			fmt.Printf("任务 %d 开始执行 \n", taskID)
			time.Sleep(time.Second * 1) // 模拟耗时操作
			fmt.Printf("任务 %d 执行完成 end\n", taskID)
		}()
	}

	// 等待所有任务完成
	wg.Wait()

	fmt.Println("所有任务已完成，程序即将退出")
}