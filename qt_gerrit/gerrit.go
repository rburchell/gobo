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
	"golang.org/x/crypto/ssh"
	"io/ioutil"
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
			Ciphers: []string{
				"aes128-cbc",
			},
		},
	}

	client, err := ssh.Dial("tcp", "codereview.qt-project.org:29418", config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	session, err := client.NewSession()
	if err != nil {
		panic("fail to create session:" + err.Error())
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		panic("Failed to get stdout pipe: " + err.Error())
	}

	bio := bufio.NewReader(stdout)
	if err := session.Start("gerrit stream-events"); err != nil {
		panic("fail to run " + err.Error())
	}
	for {
		println("Reading line")
		buffer, _, err := bio.ReadLine()
		if err != nil {
			panic("Error reading line: " + err.Error())
		}
		println("O: " + string(buffer))
	}
}
