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

package main

import "github.com/rburchell/gobo/irc/parser"
import "github.com/rburchell/gobo/irc/client"
import "fmt"

func main() {
	c := client.NewClient("qt_gerrit", "qt_gerrit", "Qt IRC Bot")

	c.AddCallback(client.OnMessage, func(c *client.IrcClient, command *parser.IrcCommand) {
		fmt.Printf("In PRIVMSG callback: %v\n", command)
	})

	c.AddCallback(client.OnConnected, func(c *client.IrcClient, command *parser.IrcCommand) {
		fmt.Printf("In CONNECTED callback: %v\n", command)
	})

	c.Join("#gobo")
	go c.Run("irc.chatspike.net:6667")

	gc := NewClient()
	go gc.Run()

	for {
		select {
		case command := <-c.CommandChannel:
			c.ProcessCallbacks(command)
		case msg := <-gc.MessageChannel:
			if msg.Type == "comment-added" {
				reviewstring := ""

				for _, approval := range msg.Approvals {
					if approval.Value < 0 {
						// dark red
					} else if approval.Value > 0 {
						// dark green
					}

					atype := approval.Type

					// newer Gerrit uses long form type strings
					// canonicalize them into something useful
					if approval.Type == "Code-Review" {
						atype = "C"
					} else if approval.Type == "Sanity-Review" {
						atype = "S"
					} else if approval.Type == "CRVW" {
						atype = "C"
					} else if approval.Type == "SRVW" {
						atype = "S"
					} else {
						panic("Unknown approval type " + atype)
					}

					// TODO: color wrap
					reviewstring += fmt.Sprintf("%s: %d", atype, approval.Value)
				}

				// TODO: handle color when we add it
				if msg.Author.Email == "qt_sanitybot@qt-project.org" && reviewstring == "S: 1" {
					// drop these, they're spammy
				} else {
					if len(msg.Approvals) > 0 {
						msg := fmt.Sprintf("[%s/%s] %s (%s) reviewed by %s: %s",
							msg.Change.Project, msg.Change.Branch,
							msg.Change.Subject, msg.Change.Url,
							msg.Author.Name, reviewstring)
						c.WriteMessage("#gobo", msg)
					} else {
						msg := fmt.Sprintf("[%s/%s] %s (%s) commented by %s",
							msg.Change.Project, msg.Change.Branch,
							msg.Change.Subject, msg.Change.Url,
							msg.Author.Name)
						c.WriteMessage("#gobo", msg)
					}
				}
			}
			println(fmt.Sprintf("Gerrit: Message: %s\n", msg.OriginalJson))
		}
	}
}
