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
