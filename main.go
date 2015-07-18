package main

import "fmt"

func main() {
    client := NewClient("gobo")

    client.AddCallback("PRIVMSG", func (client *Client, command *Command) {
        fmt.Printf("In PRIVMSG callback: %v\n", command)
    })

    go client.Run()

    for {
        select {
        case command := <- client.CommandChannel:
            fmt.Printf("MAIN: %v\n", command)
        }
    }
}

