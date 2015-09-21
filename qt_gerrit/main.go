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
import "regexp"
import "net/http"
import "io/ioutil"
import "encoding/json"

type JiraBug struct {
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`

	ErrorMessages []string `json:"errorMessages"`
}

func main() {
	c := client.NewClient("qt_gerrit", "qt_gerrit", "Qt IRC Bot")

	c.AddCallback(client.OnMessage, func(c *client.IrcClient, command *parser.IrcCommand) {
		br := regexp.MustCompile(`\b(Q[A-Z]+-[0-9]+)\b`)
		bugs := br.FindAllString(command.Parameters[1], -1)

		go func() {
			for _, bug := range bugs {
				res, err := http.Get("https://bugreports.qt.io/rest/api/2/issue/" + bug)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s (while fetching HTTP): %s", bug, err.Error()))
					continue
				}

				jsonBlob, err := ioutil.ReadAll(res.Body)
				res.Body.Close()
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s (while reading response): %s", bug, err.Error()))
					continue
				}

				var bugReport JiraBug
				err = json.Unmarshal(jsonBlob, &bugReport)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s (while parsing JSON): %s", bug, err.Error()))
					continue
				}

				if len(bugReport.ErrorMessages) == 0 {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("%s - https://bugreports.qt.io/browse/%s\n", bugReport.Fields.Summary, bug))
				} else {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s: %s", bug, bugReport.ErrorMessages[0]))
				}
			}
		}()
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
				handleCommentAdded(c, msg)
			} else if msg.Type == "patchset-created" {
				handlePatchSetCreated(c, msg)
			} else if msg.Type == "change-merged" {
				handleChangeMerged(c, msg)
			} else if msg.Type == "reviewer-added" {
				// ignore, too spammy
			} else if msg.Type == "ref-updated" {
				// ignore, too spammy
			}
			println(fmt.Sprintf("Gerrit: Message: %s\n", msg.OriginalJson))
		}
	}
}
