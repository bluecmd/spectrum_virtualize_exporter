// Tests of spectrum_virtualize_exporter
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
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
	"testing"

	"github.com/google/go-jsonnet"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type fakeClient struct {
	data map[string][]byte
}

func (c *fakeClient) prepare(path string, jfile string) {
	vm := jsonnet.MakeVM()
	b, err := ioutil.ReadFile(jfile)
	if err != nil {
		log.Fatalf("Failed to read jsonnet %q: %v", jfile, err)
	}
	output, err := vm.EvaluateSnippet(jfile, string(b))
	if err != nil {
		log.Fatalf("Failed to evaluate jsonnet %q: %v", jfile, err)
	}
	c.data[path] = []byte(output)
}

func (c *fakeClient) Get(path string, query string, obj interface{}) error {
	d, ok := c.data[path]
	if !ok {
		log.Fatalf("Tried to get unprepared URL %q", path)
	}
	return json.Unmarshal(d, obj)
}

func newFakeClient() *fakeClient {
	return &fakeClient{data: map[string][]byte{}}
}

func TestEnclosureStats(t *testing.T) {
	c := newFakeClient()
	c.prepare("rest/lsenclosurestats", "testdata/lsenclosurestats.jsonnet")
	r := prometheus.NewPedanticRegistry()
	if !probeEnclosureStats(c, r) {
		t.Errorf("probeEnclosureStats() returned non-success")
	}

	em := `
	# HELP spectrum_power_watts Current power draw of enclosure in watts
	# TYPE spectrum_power_watts gauge
	spectrum_power_watts{enclosure="1"} 427
	# HELP spectrum_temperature Current enclosure temperature in celsius
	# TYPE spectrum_temperature gauge
	spectrum_temperature{enclosure="1"} 26
	`

	if err := testutil.GatherAndCompare(r, strings.NewReader(em)); err != nil {
		t.Fatalf("metric compare: err %v", err)
	}
}
func TestDrive(t *testing.T) {
	c := newFakeClient()
	c.prepare("rest/lsdrive", "testdata/lsdrive.jsonnet")
	r := prometheus.NewPedanticRegistry()
	if !probeDrives(c, r) {
		t.Errorf("probeDrives() returned non-success")
	}

	em := `
	# HELP spectrum_drive_status Status of drive
	# TYPE spectrum_drive_status gauge
	spectrum_drive_status{enclosure="1",id="0",slot_id="5",status="degraded"} 0
	spectrum_drive_status{enclosure="1",id="0",slot_id="5",status="offline"} 0
	spectrum_drive_status{enclosure="1",id="0",slot_id="5",status="online"} 1
	spectrum_drive_status{enclosure="1",id="1",slot_id="1",status="degraded"} 1
	spectrum_drive_status{enclosure="1",id="1",slot_id="1",status="offline"} 0
	spectrum_drive_status{enclosure="1",id="1",slot_id="1",status="online"} 0
	spectrum_drive_status{enclosure="1",id="17",slot_id="8",status="degraded"} 0
	spectrum_drive_status{enclosure="1",id="17",slot_id="8",status="offline"} 0
	spectrum_drive_status{enclosure="1",id="17",slot_id="8",status="online"} 1
	`

	if err := testutil.GatherAndCompare(r, strings.NewReader(em)); err != nil {
		t.Fatalf("metric compare: err %v", err)
	}
}

func TestEnclosurePSU(t *testing.T) {
	c := newFakeClient()
	c.prepare("rest/lsenclosurepsu", "testdata/lsenclosurepsu.jsonnet")
	r := prometheus.NewPedanticRegistry()
	if !probeEnclosurePSUs(c, r) {
		t.Errorf("probeEnclosurePSUs() returned non-success")
	}

	em := `
	# HELP spectrum_psu_status Status of PSU
	# TYPE spectrum_psu_status gauge
	spectrum_psu_status{enclosure="1",id="1",status="degraded"} 0
	spectrum_psu_status{enclosure="1",id="1",status="offline"} 0
	spectrum_psu_status{enclosure="1",id="1",status="online"} 1
	spectrum_psu_status{enclosure="1",id="2",status="degraded"} 0
	spectrum_psu_status{enclosure="1",id="2",status="offline"} 0
	spectrum_psu_status{enclosure="1",id="2",status="online"} 1
	`

	if err := testutil.GatherAndCompare(r, strings.NewReader(em)); err != nil {
		t.Fatalf("metric compare: err %v", err)
	}
}

func TestPool(t *testing.T) {
	c := newFakeClient()
	c.prepare("rest/lsmdiskgrp", "testdata/lsmdiskgrp.jsonnet")
	r := prometheus.NewPedanticRegistry()
	if !probePool(c, r) {
		t.Errorf("probePool() returned non-success")
	}

	em := `
	# HELP spectrum_pool_capacity_bytes Capacity of pool in bytes
	# TYPE spectrum_pool_capacity_bytes gauge
	spectrum_pool_capacity_bytes{id="0",name="Pool0"} 1.0709243254538e+13
	# HELP spectrum_pool_free_bytes Free bytes in pool
	# TYPE spectrum_pool_free_bytes gauge
	spectrum_pool_free_bytes{id="0",name="Pool0"} 9.829633952317e+12
	# HELP spectrum_pool_status Status of pool
	# TYPE spectrum_pool_status gauge
	spectrum_pool_status{id="0",name="Pool0",status="offline"} 0
	spectrum_pool_status{id="0",name="Pool0",status="online"} 1
	# HELP spectrum_pool_used_bytes Used bytes in pool
	# TYPE spectrum_pool_used_bytes gauge
	spectrum_pool_used_bytes{id="0",name="Pool0"} 5.86252298485e+11
	# HELP spectrum_pool_volume_count Number of volumes associated with pool
	# TYPE spectrum_pool_volume_count gauge
	spectrum_pool_volume_count{id="0",name="Pool0"} 44
	`

	if err := testutil.GatherAndCompare(r, strings.NewReader(em)); err != nil {
		t.Fatalf("metric compare: err %v", err)
	}
}

func TestNodeStats(t *testing.T) {
	c := newFakeClient()
	c.prepare("rest/lsnodecanisterstats", "testdata/lsnodecanisterstats.jsonnet")
	r := prometheus.NewPedanticRegistry()
	if !probeNodeStats(c, r) {
		t.Errorf("probeNodeStats() returned non-success")
	}

	em := `
	# HELP spectrum_node_compression_usage_ratio Current ratio of allocated CPU for compresion
	# TYPE spectrum_node_compression_usage_ratio gauge
	spectrum_node_compression_usage_ratio{id="1"} 0.24
	spectrum_node_compression_usage_ratio{id="2"} 0
	# HELP spectrum_node_fc_bps Current bytes-per-second being transferred over Fibre Channel
	# TYPE spectrum_node_fc_bps gauge
	spectrum_node_fc_bps{id="1"} 1.048576e+06
	spectrum_node_fc_bps{id="2"} 0
	# HELP spectrum_node_fc_iops Current I/O-per-second being transferred over Fibre Channel
	# TYPE spectrum_node_fc_iops gauge
	spectrum_node_fc_iops{id="1"} 5
	spectrum_node_fc_iops{id="2"} 5
	# HELP spectrum_node_iscsi_bps Current bytes-per-second being transferred over iSCSI
	# TYPE spectrum_node_iscsi_bps gauge
	spectrum_node_iscsi_bps{id="1"} 0
	spectrum_node_iscsi_bps{id="2"} 0
	# HELP spectrum_node_iscsi_iops Current I/O-per-second being transferred over iSCSI
	# TYPE spectrum_node_iscsi_iops gauge
	spectrum_node_iscsi_iops{id="1"} 0
	spectrum_node_iscsi_iops{id="2"} 11
	# HELP spectrum_node_sas_bps Current bytes-per-second being transferred over backend SAS
	# TYPE spectrum_node_sas_bps gauge
	spectrum_node_sas_bps{id="1"} 0
	spectrum_node_sas_bps{id="2"} 0
	# HELP spectrum_node_sas_iops Current I/O-per-second being transferred over backend SAS
	# TYPE spectrum_node_sas_iops gauge
	spectrum_node_sas_iops{id="1"} 5
	spectrum_node_sas_iops{id="2"} 0
	# HELP spectrum_node_system_usage_ratio Current ratio of allocated CPU for system
	# TYPE spectrum_node_system_usage_ratio gauge
	spectrum_node_system_usage_ratio{id="1"} 0.01
	spectrum_node_system_usage_ratio{id="2"} 0.01
	# HELP spectrum_node_total_cache_usage_ratio Total percentage for both the write and read cache usage for the node
	# TYPE spectrum_node_total_cache_usage_ratio gauge
	spectrum_node_total_cache_usage_ratio{id="1"} 0.79
	spectrum_node_total_cache_usage_ratio{id="2"} 0.79
	# HELP spectrum_node_write_cache_usage_ratio Ratio of the write cache usage for the node
	# TYPE spectrum_node_write_cache_usage_ratio gauge
	spectrum_node_write_cache_usage_ratio{id="1"} 0.25
	spectrum_node_write_cache_usage_ratio{id="2"} 0.25
	`

	if err := testutil.GatherAndCompare(r, strings.NewReader(em)); err != nil {
		t.Fatalf("metric compare: err %v", err)
	}
}
