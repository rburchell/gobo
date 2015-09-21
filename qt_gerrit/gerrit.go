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
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
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
		Number  int64        `json:"number,string"` // 125617
		Subject string       `json:"subject"`       // Make QML composite types inherit enums
		Owner   GerritPerson `json:"owner"`
		Url     string       `json:"url"` // https://codereview.qt-project.org/125617
	} `json:"change"`
	PatchSet struct {
		Number         int64        `json:"number,string"` // 9
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

	// used in ref-updated
	Submitter GerritPerson `json:"submitter"`
	RefUpdate struct {
		OldRev  string `json:"oldRev"`
		NewRev  string `"json:"newRev"`
		RefName string `"json:refName"`
		Project string `"json:project"`
	} `json:"refname"`

	OriginalJson []byte
}

func connectToGerrit(signer *ssh.Signer, reconnectDelay *int) (*ssh.Client, *bufio.Reader) {
	// we use > 1 as a cheap trick here, we allow ourselves one
	// disconnect (by EOF) before we start delaying reconnection
	// attempts.
	//
	// this can (and probably should) be done in a more obvious fashion.
	if *reconnectDelay > 1 {
		if *reconnectDelay > 60 {
			*reconnectDelay = 60
		}

		fmt.Printf("Delaying reconnection attempt by %d seconds\n", *reconnectDelay)
		timer := time.NewTimer(time.Second * time.Duration(*reconnectDelay))
		<-timer.C
	}

	config := &ssh.ClientConfig{
		User: "w00t",
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
	}

	println("Connecting...")
	client, err := ssh.Dial("tcp", "codereview.qt-project.org:29418", config)
	if err != nil {
		println("Failed to dial: " + err.Error())
		*reconnectDelay += 4 // something is probably wrong with the server.
		return nil, nil
	}

	session, err := client.NewSession()
	if err != nil {
		println("fail to create session:" + err.Error())
		*reconnectDelay += 4
		return nil, nil
	}
	//defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		println("Failed to get stdout pipe: " + err.Error())
		*reconnectDelay += 4
		return nil, nil
	}

	bio := bufio.NewReader(stdout)
	if err := session.Start("gerrit stream-events"); err != nil {
		println("fail to run " + err.Error())
		*reconnectDelay += 10 // uh oh.
		return nil, nil
	} else {
		*reconnectDelay = 0
	}

	return client, bio
}

type GerritClient struct {
	MessageChannel chan *GerritMessage
	client         *ssh.Client
}

func NewClient() *GerritClient {
	client := new(GerritClient)
	client.MessageChannel = make(chan *GerritMessage)
	return client
}

func (this *GerritClient) Run() {
	keybytes, err := ioutil.ReadFile("/Users/burchr/.ssh/id_rsa")
	if err != nil {
		panic("Failed to read SSH key: " + err.Error())
	}

	signer, err := ssh.ParsePrivateKey(keybytes)
	if err != nil {
		panic("Failed to parse SSH key: " + err.Error())
	}

	var bio *bufio.Reader
	var reconnectDelay int
	for {
		for this.client == nil {
			this.client, bio = connectToGerrit(&signer, &reconnectDelay)
		}

		jsonBlob, _, err := bio.ReadLine()
		if err != nil {
			println("Error reading line: " + err.Error())
			this.client = nil
			bio = nil
			reconnectDelay += 1
		} else {
			var message GerritMessage
			err := json.Unmarshal(jsonBlob, &message)
			message.OriginalJson = jsonBlob
			if err != nil {
				println("BAD JSON: " + string(jsonBlob))
				panic("Error processing JSON! " + err.Error())
			}

			this.MessageChannel <- &message
		}
	}
}
