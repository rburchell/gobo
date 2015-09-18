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
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"time"
)

func Gerrit() {
	keybytes, err := ioutil.ReadFile("/Users/burchr/.ssh/id_rsa")
	if err != nil {
		panic("Failed to read SSH key: " + err.Error())
	}

	println(string(keybytes))
	signer, err := ssh.ParsePrivateKey(keybytes)
	if err != nil {
		panic("Failed to parse SSH key: " + err.Error())
	}

	config := &ssh.ClientConfig{
		User: "w00t",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		Config: ssh.Config{
			// this should be rechecked whenever Gerrit is upgraded, and ideally
			// done away with once a better cipher is available there.
			Ciphers: []string{
				"aes128-cbc",
			},
		},
	}

	var client *ssh.Client
	var bio *bufio.Reader
	var reconnDelay int
	for {
		if client == nil {
			// we use > 1 as a cheap trick here, we allow ourselves one
			// disconnect (by EOF) before we start delaying reconnection
			// attempts.
			//
			// this can (and probably should) be done in a more obvious fashion.
			if reconnDelay > 1 {
				if reconnDelay > 60 {
					reconnDelay = 60
				}

				fmt.Printf("Delaying reconnection attempt by %d seconds\n", reconnDelay)
				timer := time.NewTimer(time.Second * time.Duration(reconnDelay))
				<-timer.C
			}

			println("Connecting...")
			client, err := ssh.Dial("tcp", "codereview.qt-project.org:29418", config)
			if err != nil {
				println("Failed to dial: " + err.Error())
				reconnDelay += 4 // something is probably wrong with the server.
				client = nil
				continue
			}

			session, err := client.NewSession()
			if err != nil {
				println("fail to create session:" + err.Error())
				reconnDelay += 4
				client = nil
				continue
			}
			defer session.Close()

			stdout, err := session.StdoutPipe()
			if err != nil {
				println("Failed to get stdout pipe: " + err.Error())
				reconnDelay += 4
				client = nil
				continue
			}

			bio = bufio.NewReader(stdout)
			if err := session.Start("gerrit stream-events"); err != nil {
				println("fail to run " + err.Error())
				reconnDelay += 10 // uh oh.
				client = nil
				continue
			} else {
				reconnDelay = 0
			}
		}

		println("Reading line")
		buffer, _, err := bio.ReadLine()
		if err != nil {
			println("Error reading line: " + err.Error())
			client = nil
			bio = nil
			reconnDelay += 1
		}

		println("O: " + string(buffer))
	}
}
