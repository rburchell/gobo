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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// TODO: utterly incomplete, because this is a big response and we only care about a
// small fraction of it right now.
type JiraBug struct {
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`

	ErrorMessages []string `json:"errorMessages"`
}

func handleJiraWebApi(resultsChannel chan string, directTo string, bugs []string) {
	defer func() { close(resultsChannel) }()

	hclient := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	for _, bugId := range bugs {
		res, err := hclient.Get("https://bugreports.qt.io/rest/api/2/issue/" + bugId)
		if err != nil {
			resultsChannel <- fmt.Sprintf("Error retrieving bug %s (while fetching HTTP): %s", bugId, err.Error())
			continue
		}

		jsonBlob, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			resultsChannel <- fmt.Sprintf("Error retrieving bug %s (while reading response): %s", bugId, err.Error())
			continue
		}

		var bug JiraBug
		err = json.Unmarshal(jsonBlob, &bug)
		if err != nil {
			resultsChannel <- fmt.Sprintf("Error retrieving bug %s (while parsing JSON): %s", bugId, err.Error())
			continue
		}

		if len(bug.ErrorMessages) > 0 {
			resultsChannel <- fmt.Sprintf("Error retrieving bug %s: %s", bugId, bug.ErrorMessages[0])
			continue
		}

		if len(bug.Fields.Summary) == 0 {
			resultsChannel <- fmt.Sprintf("Error retrieving bug %s: malformed reply", bugId)
			continue
		}

		resultsChannel <- fmt.Sprintf("%s%s - https://bugreports.qt.io/browse/%s (%s)",
			directTo,
			bug.Fields.Summary,
			bugId,
			bug.Fields.Status.Name)
	}
}
