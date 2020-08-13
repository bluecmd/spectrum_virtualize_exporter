// All currently supported probes
//
// Copyright (C) 2020  Christian Svensson
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
)

func probeEnclosureStats(c SpectrumHTTP, registry *prometheus.Registry) bool {
	var (
		mPower = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_power_watts",
				Help: "Current power draw of enclosure in watts",
			},
			[]string{"enclosure"},
		)
		mTemp = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_temperature",
				Help: "Current enclosure temperature in celsius",
			},
			[]string{"enclosure"},
		)
	)

	registry.MustRegister(mPower)
	registry.MustRegister(mTemp)

	type enclosureStats struct {
		EnclosureID string `json:"enclosure_id"`
		StatName    string `json:"stat_name"`
		StatCurrent int    `json:"stat_current,string"`
	}
	var st []enclosureStats

	if err := c.Get("lsenclosurestats", "", &st); err != nil {
		log.Printf("Error: %v", err)
		return false
	}

	for _, s := range st {
		if s.StatName == "power_w" {
			mPower.WithLabelValues(s.EnclosureID).Set(float64(s.StatCurrent))
		} else if s.StatName == "temp_c" {
			mTemp.WithLabelValues(s.EnclosureID).Set(float64(s.StatCurrent))
		}
	}
	return true
}

func probe(ctx context.Context, target string, registry *prometheus.Registry, hc *http.Client) (bool, error) {
	tgt, err := url.Parse(target)
	if err != nil {
		return false, fmt.Errorf("url.Parse failed: %v", err)
	}

	if tgt.Scheme != "https" && tgt.Scheme != "http" {
		return false, fmt.Errorf("Unsupported scheme %q", tgt.Scheme)
	}

	// Filter anything else than scheme and hostname
	u := url.URL{
		Scheme: tgt.Scheme,
		Host:   tgt.Host,
	}
	c, err := newSpectrumClient(ctx, u, hc)
	if err != nil {
		return false, err
	}

	// TODO: Make parallel
	success := probeEnclosureStats(c, registry)

	return success, nil
}
