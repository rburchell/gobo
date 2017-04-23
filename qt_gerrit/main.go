/*
 * Copyright (C) 2015-2017 Robin Burchell <robin+git@viroteck.net>
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
	"fmt"
	"github.com/rburchell/gobo/irc/client"
	"github.com/rburchell/gobo/irc/parser"
	"os"
	"regexp"
	"strings"
	"time"
)

func messageDrainer(c *client.IrcClient, origin string, messageChan chan string) {
	for {
		select {
		case msg := <-messageChan:
			if len(msg) == 0 {
				return
			}

			c.WriteMessage(origin, msg)
		}
	}
}

func main() {
	// TODO: move all env var checks here.
	if len(gerritChannel) == 0 {
		panic("Must provide environment variable GERRIT_CHANNEL")
	}

	c := client.NewClient("qt_gerrit", "qt_gerrit", "Qt IRC Bot")

	c.AddCallback(client.OnMessage, func(c *client.IrcClient, command *parser.IrcMessage) {
		directRegex := regexp.MustCompile(`^([^ ]+[,:] )`)
		directTo := directRegex.FindString(command.Parameters[1]) // was this directed at someone?
		if len(directTo) == 0 {
			directTo = command.Prefix.Nick + ": " // if not, default to sender of the message
		}

		// Jira
		br := regexp.MustCompile(`\b(Q[A-Z]+-[0-9]+)\b`)
		bugs := br.FindAllString(command.Parameters[1], -1)

		jiraChan := make(chan string)
		go handleJiraWebApi(jiraChan, directTo, bugs)
		go messageDrainer(c, command.Parameters[0], jiraChan)

		// Gerrit
		cr := regexp.MustCompile(`(I[0-9a-f]{40})`)
		changes := cr.FindAllString(command.Parameters[1], -1)

		cr2 := regexp.MustCompile(`https:\/\/codereview\.qt\-project\.org\/(?:\#\/c\/)?([0-9]+|[0-9]+)\/?`)
		changes2 := cr2.FindAllStringSubmatch(command.Parameters[1], -1)

		for _, change := range changes2 {
			changes = append(changes, change[1])
		}

		gerritChan := make(chan string)
		go handleGerritWebApi(gerritChan, directTo, changes)
		go messageDrainer(c, command.Parameters[0], gerritChan)

		// Github
		commitre := regexp.MustCompile(`(` + strings.Join(validBareRepoNames, "|") + `)\/([0-9a-f]+)`)
		commitz := commitre.FindAllStringSubmatch(command.Parameters[1], -1)

		ghChan := make(chan string)
		go handleGithubWebApi(ghChan, directTo, commitz)
		go messageDrainer(c, command.Parameters[0], ghChan)
	})

	c.AddCallback(client.OnConnected, func(c *client.IrcClient, command *parser.IrcMessage) {
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
		case msg := <-gc.DiagnosticsChannel:
			str := fmt.Sprintf("[DIAGNOSTICS] %s", msg)
			c.WriteMessage(gerritChannel, str)
		case msg := <-gc.MessageChannel:
			time.Sleep(1 * time.Second) // poor man's throttling so Gerrit doesn't flood us off
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
				refUpdateChan := make(chan string)
				go handleRefUpdate(refUpdateChan, msg)
				go messageDrainer(c, gerritChannel, refUpdateChan)
			}
			println(fmt.Sprintf("Gerrit: Message: %s\n", msg.OriginalJson))
		}
	}
}
