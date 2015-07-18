package main

import "bufio"
import "net"
import "fmt"

type Client struct {
    Conn net.Conn
    CommandChannel chan *Command
    callbacks map[string][]CommandFunc
}

// An CommandFunc is a callback function to handle a received command from a
// client.
type CommandFunc func(client *Client, command *Command)()

func NewClient(nick string) *Client {
    client := new(Client)
    mchan := make(chan *Command)
    client.CommandChannel = mchan
    client.callbacks = make(map[string][]CommandFunc)

    return client
}

func (this *Client) AddCallback(command string, callback CommandFunc) {
    this.callbacks[command] = append(this.callbacks[command], callback)
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

        callbacks := this.callbacks[command.Command]

        for _, callback := range callbacks {
            callback(this, command)
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


