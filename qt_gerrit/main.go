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
import "bytes"

type GerritChange struct {
	Kind      string `json:"kind"`
	Id        string `json:"id"`
	Project   string `json:"project"`
	Branch    string `json:"branch"`
	ChangeId  string `json:"change_id"`
	Subject   string `json:"subject"`
	Status    string `json:"status"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
	Mergeable bool   `json:"mergeable"`
	SortKey   string `json:"_sortkey"`
	Number    int    `json:"_number"`
	Owner     struct {
		Name string `json:"name"`
	} `json:"owner"`
}

// TODO: utterly incomplete, because this is a big response and we only care about a
// small fraction of it right now.
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

				if len(bugReport.ErrorMessages) > 0 {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s: %s", bug, bugReport.ErrorMessages[0]))
					continue
				}

				if len(bugReport.Fields.Summary) == 0 {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s: malformed reply", bug))
					continue
				}

				c.WriteMessage(command.Parameters[0], fmt.Sprintf("%s - https://bugreports.qt.io/browse/%s\n", bugReport.Fields.Summary, bug))
			}
		}()

		cr := regexp.MustCompile(`(I[0-9a-f]{40})`)
		changes := cr.FindAllString(command.Parameters[1], -1)

		go func() {
			for _, change := range changes {
				res, err := http.Get("https://codereview.qt-project.org/changes/" + change)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (while fetching HTTP): %s", change, err.Error()))
					continue
				}

				jsonBlob, err := ioutil.ReadAll(res.Body)
				res.Body.Close()
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (while reading response): %s", change, err.Error()))
					continue
				}

				// From the Gerrit documentation:
				// To prevent against Cross Site Script Inclusion (XSSI) attacks, the JSON
				// response body starts with a magic prefix line that must be stripped before
				// feeding the rest of the response body to a JSON parser:
				//    )]}'
				//    [ ... valid JSON ... ]
				if !bytes.HasPrefix(jsonBlob, []byte(")]}'\n")) {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (couldn't find Gerrit magic)", change))
					continue
				}

				// strip off the gerrit magic
				jsonBlob = jsonBlob[5:]

				var change GerritChange
				err = json.Unmarshal(jsonBlob, &change)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (while parsing JSON): %s", change, err.Error()))
					continue
				}

				if len(change.Id) == 0 {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s: malformed reply", change))
					continue
				}

				c.WriteMessage(command.Parameters[0], fmt.Sprintf("[%s/%s] %s from %s - %s (%s)",
					change.Project, change.Branch, change.Subject, change.Owner.Name,
					fmt.Sprintf("https://codereview.qt-project.org/%s", change.Number), change.Status))
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
