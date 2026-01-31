package main

import (
    "fmt"
)

func sendToChannel(ch chan int) {
    ch <- 42 // 会修改原channel
}

func main() {
    ch := make(chan int, 1)
    sendToChannel(ch)
    fmt.Println(<-ch) // 输出: 42
}
