package main

import (
    "fmt"
    "time"
)

var count int

func main() {
    for i := 0; i < 1000; i++ {
        go add()
    }
    time.Sleep(1 * time.Second)
    fmt.Println("count:", count)
}

func add() {
    count++
}