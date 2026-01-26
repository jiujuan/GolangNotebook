package main

import (
    "fmt"
    "sync"
    "time"
)

var (
    count int
    mutex sync.Mutex
)

func main() {
    for i := 0; i < 1000; i++ {
        go add()
    }
    time.Sleep(1 * time.Second)
    fmt.Println("count:", count)
}

func add() {
    mutex.Lock()
    count++
    mutex.Unlock()
}