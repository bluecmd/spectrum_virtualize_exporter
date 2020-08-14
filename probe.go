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

	"github.com/alecthomas/units"
	"github.com/prometheus/client_golang/prometheus"
)

func probeNodeStats(c SpectrumHTTP, registry *prometheus.Registry) bool {
	var (
		mCmpCPU = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_compression_usage_ratio",
				Help: "Current ratio of allocated CPU for compresion",
			},
			[]string{"id"},
		)
		mSysCPU = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_system_usage_ratio",
				Help: "Current ratio of allocated CPU for system",
			},
			[]string{"id"},
		)
		mCacheWrite = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_write_cache_usage_ratio",
				Help: "Ratio of the write cache usage for the node",
			},
			[]string{"id"},
		)
		mCacheTotal = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_total_cache_usage_ratio",
				Help: "Total percentage for both the write and read cache usage for the node",
			},
			[]string{"id"},
		)
		mFcBytes = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_fc_bps",
				Help: "Current bytes-per-second being transferred over Fibre Channel",
			},
			[]string{"id"},
		)
		mFcIO = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_fc_iops",
				Help: "Current I/O-per-second being transferred over Fibre Channel",
			},
			[]string{"id"},
		)
		mISCSIBytes = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_iscsi_bps",
				Help: "Current bytes-per-second being transferred over iSCSI",
			},
			[]string{"id"},
		)
		mISCSIIO = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_iscsi_iops",
				Help: "Current I/O-per-second being transferred over iSCSI",
			},
			[]string{"id"},
		)
		mSASBytes = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_sas_bps",
				Help: "Current bytes-per-second being transferred over backend SAS",
			},
			[]string{"id"},
		)
		mSASIO = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_node_sas_iops",
				Help: "Current I/O-per-second being transferred over backend SAS",
			},
			[]string{"id"},
		)
	)

	registry.MustRegister(mSysCPU)
	registry.MustRegister(mCmpCPU)
	registry.MustRegister(mCacheWrite)
	registry.MustRegister(mCacheTotal)
	registry.MustRegister(mFcBytes)
	registry.MustRegister(mFcIO)
	registry.MustRegister(mISCSIBytes)
	registry.MustRegister(mISCSIIO)
	registry.MustRegister(mSASBytes)
	registry.MustRegister(mSASIO)

	type nodeStat struct {
		NodeID      string `json:"node_id"`
		StatName    string `json:"stat_name"`
		StatCurrent int    `json:"stat_current,string"`
	}
	var st []nodeStat

	if err := c.Get("rest/lsnodecanisterstats", "", &st); err != nil {
		log.Printf("Error: %v", err)
		return false
	}

	for _, s := range st {
		if s.StatName == "compression_cpu_pc" {
			mCmpCPU.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent) / 100.0)
		} else if s.StatName == "cpu_pc" {
			mSysCPU.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent) / 100.0)
		} else if s.StatName == "fc_mb" {
			mFcBytes.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent) * 1024 * 1024)
		} else if s.StatName == "fc_io" {
			mFcIO.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent))
		} else if s.StatName == "iscsi_mb" {
			mISCSIBytes.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent) * 1024 * 1024)
		} else if s.StatName == "iscsi_io" {
			mISCSIIO.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent))
		} else if s.StatName == "sas_mb" {
			mSASBytes.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent) * 1024 * 1024)
		} else if s.StatName == "sas_io" {
			mSASIO.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent))
		} else if s.StatName == "write_cache_pc" {
			mCacheWrite.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent) / 100.0)
		} else if s.StatName == "total_cache_pc" {
			mCacheTotal.WithLabelValues(s.NodeID).Set(float64(s.StatCurrent) / 100.0)
		}
	}
	return true
}

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

	if err := c.Get("rest/lsenclosurestats", "", &st); err != nil {
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

func probeDrives(c SpectrumHTTP, registry *prometheus.Registry) bool {
	labels := []string{"enclosure", "slot_id", "id"}
	var (
		mStatus = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_drive_status",
				Help: "Status of drive",
			},
			append(labels, "status"),
		)
	)

	registry.MustRegister(mStatus)

	type drive struct {
		ID          string
		Status      string
		Use         string
		Capacity    string
		SlotID      string `json:"slot_id"`
		MdiskID     string `json:"mdisk_id"`
		MdiskName   string `json:"mdisk_name"`
		EnclosureID string `json:"enclosure_id"`
	}
	var st []drive

	if err := c.Get("rest/lsdrive", "", &st); err != nil {
		log.Printf("Error: %v", err)
		return false
	}

	for _, s := range st {
		var son, soff, sdeg float64
		if s.Status == "online" {
			son = 1.0
		} else if s.Status == "offline" {
			soff = 1.0
		} else if s.Status == "degraded" {
			sdeg = 1.0
		}
		mStatus.WithLabelValues(s.EnclosureID, s.SlotID, s.ID, "online").Set(float64(son))
		mStatus.WithLabelValues(s.EnclosureID, s.SlotID, s.ID, "offline").Set(float64(soff))
		mStatus.WithLabelValues(s.EnclosureID, s.SlotID, s.ID, "degraded").Set(float64(sdeg))
	}
	return true
}

func probeEnclosurePSUs(c SpectrumHTTP, registry *prometheus.Registry) bool {
	labels := []string{"enclosure", "id"}
	var (
		mStatus = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_psu_status",
				Help: "Status of PSU",
			},
			append(labels, "status"),
		)
	)

	registry.MustRegister(mStatus)

	type psu struct {
		Status      string
		PSUID       string `json:"psu_id"`
		EnclosureID string `json:"enclosure_id"`
	}
	var st []psu

	if err := c.Get("rest/lsenclosurepsu", "", &st); err != nil {
		log.Printf("Error: %v", err)
		return false
	}

	for _, s := range st {
		var son, soff, sdeg float64
		if s.Status == "online" {
			son = 1.0
		} else if s.Status == "offline" {
			soff = 1.0
		} else if s.Status == "degraded" {
			sdeg = 1.0
		}
		mStatus.WithLabelValues(s.EnclosureID, s.PSUID, "online").Set(float64(son))
		mStatus.WithLabelValues(s.EnclosureID, s.PSUID, "offline").Set(float64(soff))
		mStatus.WithLabelValues(s.EnclosureID, s.PSUID, "degraded").Set(float64(sdeg))
	}
	return true
}

func probePool(c SpectrumHTTP, registry *prometheus.Registry) bool {
	labels := []string{"id", "name"}
	var (
		mStatus = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spectrum_pool_status",
				Help: "Status of pool",
			},
			append(labels, "status"),
		)
		mVdiskCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "spectrum_pool_volume_count", Help: "Number of volumes associated with pool"}, labels)
		mCapacity   = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "spectrum_pool_capacity_bytes", Help: "Capacity of pool in bytes"}, labels)
		mFree       = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "spectrum_pool_free_bytes", Help: "Free bytes in pool"}, labels)
		mUsed       = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "spectrum_pool_used_bytes", Help: "Used bytes in pool"}, labels)
	)

	registry.MustRegister(mStatus)
	registry.MustRegister(mVdiskCount)
	registry.MustRegister(mCapacity)
	registry.MustRegister(mFree)
	registry.MustRegister(mUsed)

	type pool struct {
		ID                  string
		Status              string
		Name                string
		VdiskCount          int `json:"vdisk_count,string"`
		Capacity            string
		FreeCapacity        string `json:"free_capacity"`
		VirtualCapacity     string `json:"virtual_capacity"`
		UsedCapacity        string `json:"used_capacity"`
		RealCapacity        string `json:"real_capacity"`
		ReclaimableCapacity string `json:"reclaimable_capacity"`
	}
	var st []pool

	if err := c.Get("rest/lsmdiskgrp", "", &st); err != nil {
		log.Printf("Error: %v", err)
		return false
	}

	for _, s := range st {
		var son, soff float64
		if s.Status == "online" {
			son = 1.0
		} else if s.Status == "offline" {
			soff = 1.0
		}
		mStatus.WithLabelValues(s.ID, s.Name, "online").Set(float64(son))
		mStatus.WithLabelValues(s.ID, s.Name, "offline").Set(float64(soff))

		mVdiskCount.WithLabelValues(s.ID, s.Name).Set(float64(s.VdiskCount))

		free, err := units.ParseBase2Bytes(s.FreeCapacity)
		if err != nil {
			log.Printf("Failed to parse %q: %v", s.FreeCapacity, err)
		} else {
			mFree.WithLabelValues(s.ID, s.Name).Set(float64(free))
		}

		capacity, err := units.ParseBase2Bytes(s.Capacity)
		if err != nil {
			log.Printf("Failed to parse %q: %v", s.Capacity, err)
		} else {
			mCapacity.WithLabelValues(s.ID, s.Name).Set(float64(capacity))
		}

		used, err := units.ParseBase2Bytes(s.UsedCapacity)
		if err != nil {
			log.Printf("Failed to parse %q: %v", s.UsedCapacity, err)
		} else {
			mUsed.WithLabelValues(s.ID, s.Name).Set(float64(used))
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
	success := probeEnclosureStats(c, registry) &&
		probeEnclosurePSUs(c, registry) &&
		probePool(c, registry) &&
		probeDrives(c, registry) &&
		probeNodeStats(c, registry)

	return success, nil
}
