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
	"fmt"
	"time"
)

type sParams struct {
	Attrs []string `json:"attrs"`
	Start int64    `json:"start"`
	End   int64    `json:"end"`
}

func (this *Client) FetchStatistics(start time.Time, end time.Time) ([]StatisticsInfo, error) {
	params := sParams{
		Attrs: []string{
			"wlan_bytes",
			"wlan-num_sta",
			"time",
		},
		Start: start.UnixNano() / 1000000,
		End:   end.UnixNano() / 1000000,
	}

	url := fmt.Sprintf("https://%s:%s/api/s/default/stat/report/5minutes.site", this.addr, this.port)
	response := &StatisticsResponse{}
	err := postInto(url, params, this.httpClient, response)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

type StatisticsResponse struct {
	Data []StatisticsInfo
	Meta struct {
		Rc string
	}
}

// ### are these data types all ok?

type StatisticsInfo struct {
	Oid  string
	Site string

	// epoch time, milliseconds
	Time int64 `json:"time"`

	WlanNumSta int64   `json:"wlan-num_sta"`
	WlanBytes  float64 `json:"wlan_bytes"`
}
