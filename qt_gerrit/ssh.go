package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"time"
)

// From http://stackoverflow.com/questions/31554196/ssh-connection-timeout

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c *Conn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

func SSHDialTimeout(network, addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	conn, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	timeoutConn := &Conn{conn, timeout, timeout}
	c, chans, reqs, err := ssh.NewClientConn(timeoutConn, addr, config)
	if err != nil {
		return nil, err
	}
	client := ssh.NewClient(c, chans, reqs)

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			<-t.C
			_, _, err := client.Conn.SendRequest("keepalive@qt_gerrit_bot", true, nil)
			if err != nil {
				fmt.Printf("Keepalive SSH failed: " + err.Error())
				client.Close()
				return
			}
		}
	}()
	return client, nil
}
