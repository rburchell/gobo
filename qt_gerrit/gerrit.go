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
	"bufio"
	"encoding/json"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
	"time"
)

type GerritPerson struct {
	Name     string `json:"name"`     // J-P Nurmi
	Email    string `json:"email"`    // jpnurmi@theqtcompany.com
	Username string `json:"username"` // jpnurmi
}

type GerritMessage struct {
	Type   string `json:"type"` // comment-added
	Change struct {
		Project string       `json:"project"`       // qt/qtdeclarative
		Branch  string       `json:"branch"`        // 5.6
		Id      string       `json:"id"`            // Icefdec91b012b12728367fd54b4d16796233ee12
		Number  int64        `json:"number"`        // 125617
		Subject string       `json:"subject"`       // Make QML composite types inherit enums
		Owner   GerritPerson `json:"owner"`
		Url     string       `json:"url"` // https://codereview.qt-project.org/125617
	} `json:"change"`
	PatchSet struct {
		Number         int64        `json:"number"`        // 9
		Revision       string       `json:"revision"`      // 52c9ebcd78379b0eacc1476237720e06abf286b3
		Parents        []string     `json:"parents"`       // ["9688aa4fe3195147881dc0969bf000bfc8a65e5e"]
		Ref            string       `json:"ref"`           // refs/changes/17/125617/9
		Uploader       GerritPerson `json:"uploader"`
		CreatedOn      int64        `json:"createdOn"` // 1442413012
		Author         GerritPerson `json:"author"`
		SizeInsertions int64        `json:"sizeInsertions"` // 80
		SizeDeletions  int64        `json:"sizeDeletions"`  // -13
	} `json:"patchSet"`
	Author    GerritPerson `json:"author"`
	Approvals []struct {
		Type        string `json:"type"`
		Description string `json:"description"`
		Value       int64  `json:"value,string"`
	} `json:"approvals"`
	Comment string `json:"comment"`

	// Used in change-abandoned
	Abandoner GerritPerson `json:"abandoner"`

	// Used in change-deferred
	Deferrer GerritPerson `json:"deferrer"`

	// used in ref-updated
	Submitter GerritPerson    `json:"submitter"`
	RefUpdate GerritRefUpdate `json:"refUpdate"`

	// used in merge-failed
	Reason string `json:"reason"`

	OriginalJson []byte
}

type GerritRefUpdate struct {
	OldRev  string `json:"oldRev"`
	NewRev  string `json:"newRev"`
	RefName string `json:"refName"`
	Project string `json:"project"`
}

// Connect to Gerrit (and keep trying until we succeed).
func (this *GerritClient) connectToGerrit(signer *ssh.Signer) (*ssh.Client, *bufio.Reader) {
	gerritUser := os.Getenv("GERRIT_USER")
	if len(gerritUser) == 0 {
		panic("Must provide GERRIT_USER environment variable.")
	}

	config := &ssh.ClientConfig{
		User: gerritUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(*signer),
		},
		Config: ssh.Config{
			// this should be rechecked whenever Gerrit is upgraded, and ideally
			// done away with once a better cipher is available there.
			Ciphers: []string{
				"aes128-cbc",
			},
		},
		HostKeyCallback: func(host string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	this.DiagnosticsChannel <- "Attempting to connect to Gerrit"

	for {
		client, err := SSHDialTimeout("tcp", "codereview.qt-project.org:29418", config, time.Second*10, time.Hour*5, time.Second*20)
		if err != nil {
			this.DiagnosticsChannel <- "Failed to dial: " + err.Error()
			time.Sleep(10 * time.Second)
			continue
		}

		session, err := client.NewSession()
		if err != nil {
			this.DiagnosticsChannel <- "Failed to create session:" + err.Error()
			time.Sleep(10 * time.Second)
			continue
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			this.DiagnosticsChannel <- "Failed to get stdout pipe: " + err.Error()
			time.Sleep(10 * time.Second)
			continue
		}

		bio := bufio.NewReader(stdout)
		if err := session.Start("gerrit stream-events"); err != nil {
			this.DiagnosticsChannel <- "Failed to stream: " + err.Error()
			time.Sleep(10 * time.Second)
			continue
		} else {
			this.DiagnosticsChannel <- "Gerrit connection reestablished."
			return client, bio
		}
	}
}

type GerritClient struct {
	MessageChannel     chan *GerritMessage
	DiagnosticsChannel chan string
	client             *ssh.Client
}

func NewClient() *GerritClient {
	client := new(GerritClient)
	client.MessageChannel = make(chan *GerritMessage)
	client.DiagnosticsChannel = make(chan string)
	return client
}

func (this *GerritClient) Run() {
	gerritKey := os.Getenv("GERRIT_PRIVATE_KEY")
	if len(gerritKey) == 0 {
		panic("Must provide GERRIT_PRIVATE_KEY environment variable.")
	}

	keybytes, err := ioutil.ReadFile(gerritKey)
	if err != nil {
		panic("Failed to read SSH key: " + err.Error())
	}

	signer, err := ssh.ParsePrivateKey(keybytes)
	if err != nil {
		panic("Failed to parse SSH key: " + err.Error())
	}

	var bio *bufio.Reader

	for {
		if this.client == nil {
			this.client, bio = this.connectToGerrit(&signer)
		}

		jsonBlob, err := bio.ReadBytes('\n')
		if err != nil {
			this.DiagnosticsChannel <- "Error reading line: " + err.Error()
			this.client.Close()
			this.client = nil
			continue // go reconnect, via connectToGerrit
		}

		var message GerritMessage
		err = json.Unmarshal(jsonBlob, &message)
		message.OriginalJson = jsonBlob
		if err != nil {
			this.DiagnosticsChannel <- "Error processing JSON: " + err.Error()
			println("BAD JSON: " + string(jsonBlob))
			continue
		}

		this.MessageChannel <- &message
	}
}
