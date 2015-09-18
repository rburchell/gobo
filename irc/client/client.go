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
import "time"

type IrcClient struct {
	conn            net.Conn
	CommandChannel  chan *parser.IrcCommand
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
type CommandFunc func(client *IrcClient, command *parser.IrcCommand)

func NewClient(nick string, user string, realname string) *IrcClient {
	client := new(IrcClient)
	mchan := make(chan *parser.IrcCommand)
	client.CommandChannel = mchan
	client.callbacks = make(map[string][]CommandFunc)
	client.nick = nick
	client.user = user
	client.realname = realname

	return client
}

func (this *IrcClient) AddCallback(command string, callback CommandFunc) {
	this.callbacks_mutex.Lock()
	this.callbacks[command] = append(this.callbacks[command], callback)
	this.callbacks_mutex.Unlock()
}

func (this *IrcClient) Run(host string) {
	var bio *bufio.Reader
	var reconnDelay int

	for {
		var buffer []byte

		for this.conn == nil || len(buffer) == 0 {
			var err error
			if this.conn == nil {
				this.connected = false
				if reconnDelay > 0 {
					if reconnDelay > 60 {
						reconnDelay = 60
					}

					fmt.Printf("Delaying reconnection attempt by %d seconds\n", reconnDelay)
					timer := time.NewTimer(time.Second * time.Duration(reconnDelay))
					<-timer.C
				}

				this.conn, err = net.Dial("tcp", host)

				if err != nil {
					// if we're having trouble connecting at all, the problem is
					// likely not a transient one (e.g. catestrophic server
					// failure, DNS failure, ...) so increase the timeout
					// quicker.
					println("Error connecting: " + err.Error())
					reconnDelay += 2
				} else {
					this.WriteLine(fmt.Sprintf("NICK %s", this.nick))
					this.WriteLine(fmt.Sprintf("USER %s * * :%s", this.user, this.realname))
					bio = bufio.NewReader(this.conn)
				}
			}

			// connection attempt may not have succeeded yet...
			if this.conn != nil {
				buffer, _, err = bio.ReadLine()
				if err != nil {
					println("Error reading line: " + err.Error())
					reconnDelay += 1
					bio = nil
					this.conn = nil
				}
			}
		}

		bufstring := string(buffer)
		println("IN: ", bufstring)
		command := parser.ParseLine(bufstring)

		switch command.Command {
		case "PING":
			this.WriteLine(fmt.Sprintf("PONG :%s", command.Parameters[0]))
		case "ERROR":
			// something is probably very wrong (e.g. a ban/kill)
			// wait a while longer to reconnect because of this
			reconnDelay += 8
		case OnConnected:
			// only reset delay on a full, successful connection. if we're
			// banned, we'll successfully establish a socket connection, but
			// there's no sense in hammering the server with reconnect attempts.
			reconnDelay = 0
			this.handleConnected()
		case OnKick:
			for _, channel := range this.irc_channels {
				if channel == command.Parameters[0] {
					if command.Parameters[1] == this.nick {
						this.WriteLine(fmt.Sprintf("JOIN %s", channel))
					}
					break
				}
			}
		}

		this.CommandChannel <- command
	}
}

func (this *IrcClient) Join(channel string) {
	this.irc_channels = append(this.irc_channels, channel)
}

var OnConnected string = "001"
var OnKick string = "KICK"
var OnMessage string = "PRIVMSG"
var OnNotice string = "NOTICE"
var OnJoin string = "JOIN"
var OnPart string = "PART"

func (this *IrcClient) ProcessCallbacks(c *parser.IrcCommand) {
	this.callbacks_mutex.Lock()
	callbacks := this.callbacks[c.Command]
	this.callbacks_mutex.Unlock()
	for _, callback := range callbacks {
		callback(this, c)
	}
}

func (this *IrcClient) WriteLine(bytes string) {
	if this.conn == nil {
		return
	}

	println("OUT: ", bytes)
	this.conn.Write([]byte(bytes))
	this.conn.Write([]byte("\r\n"))
}

func (this *IrcClient) handleConnected() {
	this.connected = true
	for _, channel := range this.irc_channels {
		this.WriteLine(fmt.Sprintf("JOIN %s", channel))
	}
}
