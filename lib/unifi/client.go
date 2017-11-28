/*
 * Copyright (C) 2017 Robin Burchell <robin+git@viroteck.net>
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

package unifi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
)

type Client struct {
	httpClient *http.Client
	addr       string
	port       string
}

func NewClient() Client {
	c := Client{}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	jar, _ := cookiejar.New(nil)
	c.httpClient = &http.Client{
		Transport: transport,
		Jar:       jar,
	}
	return c
}

func (this *Client) Login(addr string, port string, user string, pass string) error {
	this.addr = addr
	this.port = port

	url := fmt.Sprintf("https://%s:%s/api/login", addr, port)
	auth := map[string]string{
		"username": user,
		"password": pass,
	}
	return postInto(url, auth, this.httpClient, nil)
}

func getInto(url string, httpClient *http.Client, mytype interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, mytype)
	if err != nil {
		return err
	}
	return nil
}

func postInto(url string, pmap interface{}, httpClient *http.Client, mytype interface{}) error {
	paramJson, _ := json.Marshal(pmap)
	params := bytes.NewReader(paramJson)
	req, err := http.NewRequest("POST", url, params)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Not a successful request: %d (to %s)", resp.StatusCode, url))
	}

	defer resp.Body.Close()

	if mytype != nil {
		body, _ := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, mytype)
	}

	return nil
}
