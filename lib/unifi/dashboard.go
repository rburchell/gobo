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
)

func (this *Client) FetchDashboard() ([]DashboardInfo, error) {
	url := fmt.Sprintf("https://%s:%s/api/s/default/stat/dashboard?scale=5minutes", this.addr, this.port)
	response := &DashboardResponse{}
	err := getInto(url, this.httpClient, response)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

type DashboardResponse struct {
	Data []DashboardInfo
	Meta struct {
		Rc string
	}
}

// ### are these data types all ok?

type DashboardInfo struct {
	LatencyAvg *float64 `json:"latency_avg"`
	LatencyMax float64  `json:"latency_max"`
	LatencyMin float64  `json:"latency_min"`
	MaxBytesRx float64  `json:"max_rx_bytes-r"`
	MaxBytesTx float64  `json:"max_tx_bytes-r"`
	RxBytes    int64    `json:"rx_bytes-r"`

	// epoch time, milliseconds
	Time       int64   `json:"time"`
	TxBytes    int64   `json:"tx_bytes-r"`
	WanRxBytes float64 `json:"wan-rx_bytes"`
	WanTxBytes float64 `json:"wan-tx_bytes"`
}
