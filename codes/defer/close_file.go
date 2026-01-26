package main

import (
    "fmt"
    "os"
)

func writeFile(filename string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close() // 确保文件被关闭
    
    _, err = file.WriteString("Hello, defer!")
    if err != nil {
        return err
    }
    
    return nil
}

func main() {
    filename := "./a.txt"
    writeFile(filename)
    fmt.Println("End!")
}