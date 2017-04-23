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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

func handleGerritWebApi(resultsChannel chan string, directTo string, changes []string) {
	defer func() { close(resultsChannel) }()

	hclient := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	for _, changeId := range changes {
		res, err := hclient.Get("https://codereview.qt-project.org/changes/" + changeId)
		if err != nil {
			resultsChannel <- fmt.Sprintf("Error retrieving change %s (while fetching HTTP): %s", changeId, err.Error())
			continue
		}

		jsonBlob, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			resultsChannel <- fmt.Sprintf("Error retrieving change %s (while reading response): %s", changeId, err.Error())
			continue
		}

		// From the Gerrit documentation:
		// To prevent against Cross Site Script Inclusion (XSSI) attacks, the JSON
		// response body starts with a magic prefix line that must be stripped before
		// feeding the rest of the response body to a JSON parser:
		//    )]}'
		//    [ ... valid JSON ... ]
		if !bytes.HasPrefix(jsonBlob, []byte(")]}'\n")) {
			resultsChannel <- fmt.Sprintf("Error retrieving change %s (couldn't find Gerrit magic)", changeId)
			continue
		}

		// strip off the gerrit magic
		jsonBlob = jsonBlob[5:]

		var change GerritChange
		err = json.Unmarshal(jsonBlob, &change)
		if err != nil {
			resultsChannel <- fmt.Sprintf("Error retrieving change %s (while parsing JSON): %s", changeId, err.Error())
			continue
		}

		if len(change.Id) == 0 {
			resultsChannel <- fmt.Sprintf("Error retrieving change %s: malformed reply", changeId)
			continue
		}

		resultsChannel <- fmt.Sprintf("%s[%s/%s] %s from %s - %s (%s)",
			directTo, change.Project, change.Branch, change.Subject, change.Owner.Name,
			fmt.Sprintf("https://codereview.qt-project.org/%d", change.Number), change.Status)
	}
}
