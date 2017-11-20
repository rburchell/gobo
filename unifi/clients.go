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

func (this *Client) FetchDevices() ([]UnifiClient, error) {
	url := fmt.Sprintf("https://%s:%s/api/s/default/stat/sta", this.addr, this.port)
	response := &ClientResponse{}
	err := fetchInto(url, this.httpClient, response)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

type ClientResponse struct {
	Data []UnifiClient
	Meta struct {
		Rc string
	}
}

// DpiStat is for deep packet inspection stats
type DpiStat struct {
	App       int64
	Cat       int64
	RxBytes   int64
	RxPackets int64
	TxBytes   int64
	TxPackets int64
}

type UnifiClient struct {
	ID                  string `json:"_id"`
	IsGuestByUAP        bool   `json:"_is_guest_by_uap"`
	IsGuestByUGW        bool   `json:"_is_guest_by_ugw"`
	LastSeenByUAP       int64  `json:"_last_seen_by_uap"`
	LastSeenByUGW       int64  `json:"_last_seen_by_ugw"`
	UptimeByUAP         int64  `json:"_uptime_by_uap"`
	UptimeByUGW         int64  `json:"_uptime_by_ugw"`
	ApMac               string `json:"ap_mac"`
	AssocTime           int64  `json:"assoc_time"`
	Authorized          bool
	Bssid               string
	BytesR              int64 `json:"bytes-r"`
	Ccq                 int64
	Channel             int64
	DpiStats            []DpiStat `json:"dpi_stats"`
	DpiStatsLastUpdated int64     `json:"dpi_stats_last_updated"`
	Essid               string
	FirstSeen           int64  `json:"first_seen"`
	FixedIP             string `json:"fixed_ip"`
	Hostname            string
	GwMac               string `json:"gw_mac"`
	IdleTime            int64  `json:"idle_time"`
	Ip                  string
	IsGuest             bool  `json:"is_guest"`
	IsWired             bool  `json:"is_wired"`
	LastSeen            int64 `json:"last_seen"`
	LatestAssocTime     int64 `json:"latest_assoc_time"`
	Mac                 string
	Name                string
	Network             string
	NetworkID           string `json:"network_id"`
	Noise               int64
	Oui                 string
	PowersaveEnabled    bool `json:"powersave_enabled"`
	QosPolicyApplied    bool `json:"qos_policy_applied"`
	Radio               string
	RadioProto          string `json:"radio_proto"`
	RoamCount           int64  `json:"roam_count"`
	Rssi                int64
	RxBytes             int64 `json:"rx_bytes"`
	RxBytesR            int64 `json:"rx_bytes-r"`
	RxPackets           int64 `json:"rx_packets"`
	RxRate              int64 `json:"rx_rate"`
	Signal              int64
	SiteID              string `json:"site_id"`
	TxBytes             int64  `json:"tx_bytes"`
	TxBytesR            int64  `json:"tx_bytes-r"`
	TxPackets           int64  `json:"tx_packets"`
	TxPower             int64  `json:"tx_power"`
	TxRate              int64  `json:"tx_rate"`
	Uptime              int64
	UserID              string `json:"user_id"`
	Vlan                int64
}
