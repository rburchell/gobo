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

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rburchell/gobo/irc/client"
	"github.com/rburchell/gobo/irc/parser"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
)

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
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`

	ErrorMessages []string `json:"errorMessages"`
}

func main() {
	// TODO: move all env var checks here.
	if len(gerritChannel) == 0 {
		panic("Must provide environment variable GERRIT_CHANNEL")
	}

	c := client.NewClient("qt_gerrit", "qt_gerrit", "Qt IRC Bot")

	c.AddCallback(client.OnMessage, func(c *client.IrcClient, command *parser.IrcCommand) {
		directRegex := regexp.MustCompile(`^([^ ]+[,:] )`)
		directTo := directRegex.FindString(command.Parameters[1]) // was this directed at someone?
		if len(directTo) == 0 {
			directTo = command.Prefix.Nick + ": " // if not, default to sender of the message
		}

		br := regexp.MustCompile(`\b(Q[A-Z]+-[0-9]+)\b`)
		bugs := br.FindAllString(command.Parameters[1], -1)

		go func() {
			hclient := http.Client{
				Timeout: time.Duration(4 * time.Second),
			}

			for _, bugId := range bugs {
				res, err := hclient.Get("https://bugreports.qt.io/rest/api/2/issue/" + bugId)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s (while fetching HTTP): %s", bugId, err.Error()))
					continue
				}

				jsonBlob, err := ioutil.ReadAll(res.Body)
				res.Body.Close()
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s (while reading response): %s", bugId, err.Error()))
					continue
				}

				var bug JiraBug
				err = json.Unmarshal(jsonBlob, &bug)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s (while parsing JSON): %s", bugId, err.Error()))
					continue
				}

				if len(bug.ErrorMessages) > 0 {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s: %s", bugId, bug.ErrorMessages[0]))
					continue
				}

				if len(bug.Fields.Summary) == 0 {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving bug %s: malformed reply", bugId))
					continue
				}

				c.WriteMessage(command.Parameters[0], fmt.Sprintf("%s%s - https://bugreports.qt.io/browse/%s (%s)",
					directTo,
					bug.Fields.Summary,
					bugId,
					bug.Fields.Status.Name))
			}
		}()

		cr := regexp.MustCompile(`(I[0-9a-f]{40})`)
		changes := cr.FindAllString(command.Parameters[1], -1)

		go func() {
			hclient := http.Client{
				Timeout: time.Duration(4 * time.Second),
			}

			for _, changeId := range changes {
				res, err := hclient.Get("https://codereview.qt-project.org/changes/" + changeId)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (while fetching HTTP): %s", changeId, err.Error()))
					continue
				}

				jsonBlob, err := ioutil.ReadAll(res.Body)
				res.Body.Close()
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (while reading response): %s", changeId, err.Error()))
					continue
				}

				// From the Gerrit documentation:
				// To prevent against Cross Site Script Inclusion (XSSI) attacks, the JSON
				// response body starts with a magic prefix line that must be stripped before
				// feeding the rest of the response body to a JSON parser:
				//    )]}'
				//    [ ... valid JSON ... ]
				if !bytes.HasPrefix(jsonBlob, []byte(")]}'\n")) {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (couldn't find Gerrit magic)", changeId))
					continue
				}

				// strip off the gerrit magic
				jsonBlob = jsonBlob[5:]

				var change GerritChange
				err = json.Unmarshal(jsonBlob, &change)
				if err != nil {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s (while parsing JSON): %s", changeId, err.Error()))
					continue
				}

				if len(change.Id) == 0 {
					c.WriteMessage(command.Parameters[0], fmt.Sprintf("Error retrieving change %s: malformed reply", changeId))
					continue
				}

				c.WriteMessage(command.Parameters[0], fmt.Sprintf("%s[%s/%s] %s from %s - %s (%s)",
					directTo, change.Project, change.Branch, change.Subject, change.Owner.Name,
					fmt.Sprintf("https://codereview.qt-project.org/%d", change.Number), change.Status))
			}
		}()
	})

	c.AddCallback(client.OnConnected, func(c *client.IrcClient, command *parser.IrcCommand) {
		fmt.Printf("In CONNECTED callback: %v\n", command)

		nsUser := os.Getenv("NICKSERV_USER")
		if len(nsUser) == 0 {
			panic("Must provide environment variable NICKSERV_USER")
		}

		nsPass := os.Getenv("NICKSERV_PASS")
		if len(nsPass) == 0 {
			panic("Must provide environment variable NICKSERV_PASS")
		}

		c.WriteLine(fmt.Sprintf("NS IDENTIFY %s %s", nsUser, nsPass))
	})

	ircServer := os.Getenv("IRC_SERVER")
	if len(ircServer) == 0 {
		panic("Must provide environment variable IRC_SERVER")
	}

	ircChannels := os.Getenv("IRC_CHANNELS")
	if len(ircChannels) == 0 {
		panic("Must provide environment variable IRC_CHANNELS")
	}

	c.Join(ircChannels)
	go c.Run(ircServer)

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
			} else if msg.Type == "merge-failed" {
				handleMergeFailed(c, msg)
			} else if msg.Type == "reviewer-added" {
				// ignore, too spammy
			} else if msg.Type == "ref-updated" {
				// ignore, too spammy
			}
			println(fmt.Sprintf("Gerrit: Message: %s\n", msg.OriginalJson))
		}
	}
}
