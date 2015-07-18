package main

import "fmt"

func main() {
    client := NewClient("gobo")
    go client.Run()

    for {
        select {
        case command := <- client.CommandChannel:
            fmt.Printf("MAIN: %v\n", command)
        }
    }
}

