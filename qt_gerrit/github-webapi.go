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

package main

import (
	"encoding/json"
	"fmt"
	"github.com/rburchell/gobo/irc/client"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type GithubCommit struct {
	HtmlUrl string `json:"html_url"`
	Commit  struct {
		Author struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
	} `json:"commit"`
}

func handleGithubWebApi(c *client.IrcClient, origin string, directTo string, repoMap map[string]string, commitz [][]string) {
	hclient := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	// commitz:
	// [1] is the repo (e.g. qtbase)
	// [2] is the sha
	for _, repoAndSha := range commitz {
		repo := repoAndSha[1]
		sha := repoAndSha[2]

		var githubLookup string
		var ok bool
		if githubLookup, ok = repoMap[repo]; !ok {
			// sorry, not found. alter githubWhitelist.
			c.WriteMessage(origin, fmt.Sprintf("I don't know where to find commit %s in repository %s", sha, repo))
			continue
		}

		res, err := hclient.Get("https://api.github.com/repos/" + githubLookup + "/commits/" + sha)
		if err != nil {
			c.WriteMessage(origin, fmt.Sprintf("Error retrieving commit %s from repository %s (while fetching HTTP): %s", sha, repo, err.Error()))
			continue
		}

		jsonBlob, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			c.WriteMessage(origin, fmt.Sprintf("Error retrieving commit %s from repository %s (while reading response): %s", sha, repo, err.Error()))
			continue
		}

		var commit GithubCommit
		err = json.Unmarshal(jsonBlob, &commit)
		if err != nil {
			c.WriteMessage(origin, fmt.Sprintf("Error retrieving commit %s from repository %s (while parsing JSON): %s", sha, repo, err.Error()))
			continue
		}

		idx := strings.Index(commit.Commit.Message, "\n")
		if idx < 0 {
			idx = len(commit.Commit.Message)
		}
		summary := commit.Commit.Message[0:idx]

		c.WriteMessage(origin, fmt.Sprintf("%s[%s] %s from %s - %s",
			directTo, repo, summary, commit.Commit.Author.Name,
			commit.HtmlUrl))
	}
}
