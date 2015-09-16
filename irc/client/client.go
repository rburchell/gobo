/*
 * Copyright (C) 2015 Robin Burchell <robin+git@viroteck.net>
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  - Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *  - Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND ANY
 * EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
 * THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package client

import "github.com/rburchell/gobo/irc/parser"
import "bufio"
import "net"
import "fmt"
import "sync"

type Client struct {
	conn            net.Conn
	CommandChannel  chan *parser.Command
	callbacks       map[string][]CommandFunc
	callbacks_mutex sync.Mutex
	nick            string
	user            string
	realname        string
	irc_channels    []string
	connected       bool
}

// An CommandFunc is a callback function to handle a received command from a
// client.
type CommandFunc func(client *Client, command *parser.Command)

func NewClient(nick string, user string, realname string) *Client {
	client := new(Client)
	mchan := make(chan *parser.Command)
	client.CommandChannel = mchan
	client.callbacks = make(map[string][]CommandFunc)
	client.nick = nick
	client.user = user
	client.realname = realname

	return client
}

func (this *Client) AddCallback(command string, callback CommandFunc) {
	this.callbacks_mutex.Lock()
	this.callbacks[command] = append(this.callbacks[command], callback)
	this.callbacks_mutex.Unlock()
}

func (this *Client) Run(host string) {
	var bio *bufio.Reader
	for {
		var buffer []byte

		for this.conn == nil || len(buffer) == 0 {
			var err error
			if this.conn == nil {
				this.conn, err = net.Dial("tcp", host)

				if err != nil {
					panic(fmt.Sprintf("Couldn't connect to server: %s", err))
				}

				this.WriteLine(fmt.Sprintf("NICK %s", this.nick))
				this.WriteLine(fmt.Sprintf("USER %s * * :%s", this.user, this.realname))
				bio = bufio.NewReader(this.conn)
			}

			buffer, _, err = bio.ReadLine()
			if err != nil {
				println("Error reading line: " + err.Error())
				bio = nil
				this.conn = nil
			}
		}

		bufstring := string(buffer)
		println("IN: ", bufstring)
		command := parser.ParseLine(bufstring)

		switch command.Command {
		case "PING":
			this.WriteLine(fmt.Sprintf("PONG :%s", command.Parameters[0]))
		case OnConnected:
			this.handleConnected()
		default:
			this.CommandChannel <- command
		}
	}
}

func (this *Client) Join(channel string) {
	this.irc_channels = append(this.irc_channels, channel)
}

var OnConnected string = "001"

func (this *Client) ProcessCallbacks(c *parser.Command) {
	this.callbacks_mutex.Lock()
	callbacks := this.callbacks[c.Command]
	this.callbacks_mutex.Unlock()
	for _, callback := range callbacks {
		callback(this, c)
	}
}

func (this *Client) WriteLine(bytes string) {
	println("OUT: ", bytes)
	this.conn.Write([]byte(bytes))
	this.conn.Write([]byte("\r\n"))
}

func (this *Client) handleConnected() {
	this.connected = true
	for _, channel := range this.irc_channels {
		this.WriteLine(fmt.Sprintf("JOIN %s", channel))
	}
}
