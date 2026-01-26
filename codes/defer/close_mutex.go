package main

import (
    "fmt"
    "sync"
)

type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock() // 确保锁被释放
    
    c.count++
    
}

func (c *SafeCounter) GetCount() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    return c.count
}

func main() {
    sc := &SafeCounter{}
    sc.Increment()
    fmt.Println(sc.GetCount())
}