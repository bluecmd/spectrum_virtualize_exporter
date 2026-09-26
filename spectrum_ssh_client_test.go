// Tests of the SSH client
//
// Copyright (C) 2026  Christian Svensson
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
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// cliClient serves `lsfoo -delim ,` output captured from a V7000 at 8.4
// through the same parsing as spectrumSSHClient.
type cliClient struct{}

func (cliClient) Get(path string, query string, obj interface{}) error {
	b, err := os.ReadFile("testdata/" + strings.TrimPrefix(path, "rest/") + ".cli")
	if err != nil {
		return err
	}
	rows, err := parseCLITable(b)
	if err != nil {
		return err
	}
	j, _ := json.Marshal(rows)
	return json.Unmarshal(j, obj)
}

func TestParseCLITable(t *testing.T) {
	rows, err := parseCLITable([]byte("id,name,mac\n0,Pool0,40:f2:e9:00:00:01\n1,,\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0]["name"] != "Pool0" || rows[0]["mac"] != "40:f2:e9:00:00:01" || rows[1]["name"] != "" {
		t.Errorf("unexpected rows %v", rows)
	}

	rows, err = parseCLITable(nil)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Errorf("empty output: got %v, %v; want an empty list", rows, err)
	}

	if _, err := parseCLITable([]byte("a,b\n1,2,3\n")); err == nil {
		t.Errorf("row wider than the header was accepted")
	}
}

func TestSSHCommandFilter(t *testing.T) {
	c := &spectrumSSHClient{}
	for _, p := range []string{"rest/svctask rmvdisk 0", "rest/lsvdisk;rmvdisk", "rest/mkvdisk", "rest/lsvdisk -bytes"} {
		if err := c.Get(p, "", nil); err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Get(%q) was not refused: %v", p, err)
		}
	}
}

func TestPoolCLI(t *testing.T) {
	r := prometheus.NewPedanticRegistry()
	if !probePool(cliClient{}, r) {
		t.Fatalf("probePool() returned non-success")
	}
	// Pool0 is a data reduction parent pool, Pool0_CSI a quotaless child.
	em := `
	# HELP spectrum_pool_used_after_reduction_bytes Data stored in a data reduction pool, after compression and deduplication
	# TYPE spectrum_pool_used_after_reduction_bytes gauge
	spectrum_pool_used_after_reduction_bytes{id="0",name="Pool0"} 4.66192930177e+12
	spectrum_pool_used_after_reduction_bytes{id="1",name="Pool0_CSI"} 0
	# HELP spectrum_pool_used_before_reduction_bytes Data written to a data reduction pool, before compression and deduplication
	# TYPE spectrum_pool_used_before_reduction_bytes gauge
	spectrum_pool_used_before_reduction_bytes{id="0",name="Pool0"} 6.827967208488e+12
	spectrum_pool_used_before_reduction_bytes{id="1",name="Pool0_CSI"} 0
	`
	if err := testutil.GatherAndCompare(r, strings.NewReader(em),
		"spectrum_pool_used_before_reduction_bytes", "spectrum_pool_used_after_reduction_bytes"); err != nil {
		t.Fatalf("metric compare: err %v", err)
	}
	if n := testutil.CollectAndCount(r, "spectrum_pool_capacity_bytes"); n != 2 {
		t.Errorf("got %d pool capacities, want 2", n)
	}
}

func TestSystemStats(t *testing.T) {
	r := prometheus.NewPedanticRegistry()
	if !probeSystemStats(cliClient{}, r) {
		t.Fatalf("probeSystemStats() returned non-success")
	}
	// Three layers, read and write, for each of the three units.
	for _, m := range []string{"spectrum_system_iops", "spectrum_system_bytes_per_second", "spectrum_system_latency_seconds"} {
		if n := testutil.CollectAndCount(r, m); n != 6 {
			t.Errorf("%s: got %d series, want 6", m, n)
		}
	}
}
