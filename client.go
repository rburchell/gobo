package main

import "bufio"
import "net"
import "fmt"

type Client struct {
    Conn net.Conn
    CommandChannel chan *Command
}

func NewClient(nick string) *Client {
    client := new(Client)
    mchan := make(chan *Command)
    client.CommandChannel = mchan

    return client
}

func (this *Client) Run() {
    conn, err := net.Dial("tcp", "irc.chatspike.net:6667")

    if (err != nil) {
        panic(fmt.Sprintf("Couldn't connect to server", err))
    }

    this.Conn = conn
    this.WriteLine("NICK gobo")
    this.WriteLine("USER gobo * * :General Purpose IRC Bot")
    this.WriteLine("JOIN #coding")


    bio := bufio.NewReader(conn)
    for {
        buffer, _, err := bio.ReadLine()
        if (err != nil) {
            panic("Error reading")
        }

        bufstring := string(buffer)
        println("RUN: ", bufstring)

        command := ParseLine(bufstring)

        switch command.Command {
            case "PING":
                this.handlePing(command)
            default:
                this.CommandChannel <- command
        }
    }
}

func (this *Client) WriteLine(bytes string) {
    println("OUT: ", bytes)
    this.Conn.Write([]byte(bytes))
    this.Conn.Write([]byte("\r\n"))
}

func (this *Client) handlePing(c *Command) {
    this.WriteLine(fmt.Sprintf("PONG :%s", c.Parameters[0]))
}


