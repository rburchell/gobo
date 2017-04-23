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
	"fmt"
	"github.com/rburchell/gobo/irc/client"
	"os"
	"strings"
)

var gerritChannel = os.Getenv("GERRIT_CHANNEL")

func handleCommentAdded(c *client.IrcClient, msg *GerritMessage) {
	reviewstring := ""

	for _, approval := range msg.Approvals {
		if len(reviewstring) > 0 {
			reviewstring += " "
		}

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
			msg := fmt.Sprintf("[%s/%s] %s from %s reviewed by %s: %s - %s",
				msg.Change.Project, msg.Change.Branch,
				msg.Change.Subject, msg.PatchSet.Uploader.Name,
				msg.Author.Name, reviewstring, msg.Change.Url)
			c.WriteMessage(gerritChannel, msg)
		} else {
			msg := fmt.Sprintf("[%s/%s] %s from %s commented by %s - %s",
				msg.Change.Project, msg.Change.Branch,
				msg.Change.Subject, msg.PatchSet.Uploader.Name,
				msg.Author.Name, msg.Change.Url)
			c.WriteMessage(gerritChannel, msg)
		}
	}
}

func handlePatchSetCreated(c *client.IrcClient, msg *GerritMessage) {
	if msg.PatchSet.Number == 1 {
		msg := fmt.Sprintf("[%s/%s] %s pushed by %s - %s",
			msg.Change.Project, msg.Change.Branch,
			msg.Change.Subject, msg.PatchSet.Uploader.Name,
			msg.Change.Url)
		c.WriteMessage(gerritChannel, msg)
	} else {
		// TODO: msg.Owner.Name != msg.PatchSet.Uploader.Name, note
		// separately since someone else updating a patch is
		// significant
		msg := fmt.Sprintf("[%s/%s] %s updated by %s - %s",
			msg.Change.Project, msg.Change.Branch,
			msg.Change.Subject, msg.PatchSet.Uploader.Name,
			msg.Change.Url)
		c.WriteMessage(gerritChannel, msg)
	}
}

func handleChangeMerged(c *client.IrcClient, msg *GerritMessage) {
	// TODO: msg.Owner.Name != msg.PatchSet.Uploader.Name, note
	// separately since someone else updating a patch is
	// significant
	str := fmt.Sprintf("[%s/%s] %s authored by %s was cherry-picked by %s - %s",
		msg.Change.Project, msg.Change.Branch,
		msg.Change.Subject, msg.PatchSet.Uploader.Name,
		msg.Submitter.Name,
		msg.Change.Url)
	c.WriteMessage(gerritChannel, str)
}

func handleMergeFailed(c *client.IrcClient, msg *GerritMessage) {
	reasons := strings.Split(msg.Reason, "\n")
	reason := reasons[0]
	str := fmt.Sprintf("[%s/%s] %s tried to cherry-pick %s, but the merge failed because: %s - %s",
		msg.Change.Project, msg.Change.Branch,
		msg.Submitter.Name, msg.Change.Subject,
		reason, msg.Change.Url)
	c.WriteMessage(gerritChannel, str)
}

// ### It would be nice if we could actually describe *what* changed.
func handleRefUpdate(resultsChannel chan string, m *GerritMessage) {
	defer func() { close(resultsChannel) }()

	// These are handled by handleChangeMerged
	if strings.HasPrefix(m.RefUpdate.RefName, "refs/staging/") {
		return
	}

	url := "https://code.qt.io/cgit/" + m.RefUpdate.Project + ".git/log/?qt=range&q=" + m.RefUpdate.OldRev + "..." + m.RefUpdate.NewRev
	resultsChannel <- fmt.Sprintf("[%s/%s] updated from %s to %s - %s", m.RefUpdate.Project, m.RefUpdate.RefName, m.RefUpdate.OldRev, m.RefUpdate.NewRev, url)
}
