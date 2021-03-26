// Server executable of spectrum_virtualize_exporter
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
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

var (
	authMapFile    = flag.String("auth-file", "", "file containing the authentication map to use when connecting to a Spectrum Virtualize device")
	listen         = flag.String("listen", ":9747", "address to listen on")
	timeoutSeconds = flag.Int("scrape-timeout", 30, "max seconds to allow a scrape to take")
	insecure       = flag.Bool("insecure", false, "Allow insecure certificates")
	extraCAs       = flag.String("extra-ca-cert", "", "file containing extra PEMs to add to the CA trust store")

	authMap = map[string]Auth{}
)

type Auth struct {
	User     string
	Password string
}

type SpectrumHTTP interface {
	Get(path string, query string, obj interface{}) error
}

func newSpectrumClient(ctx context.Context, tgt url.URL, hc *http.Client) (SpectrumHTTP, error) {
	auth, ok := authMap[tgt.String()]
	if !ok {
		return nil, fmt.Errorf("No API authentication registered for %q", tgt.String())
	}

	if auth.User != "" && auth.Password != "" {
		c, err := newSpectrumPasswordClient(ctx, tgt, hc, auth.User, auth.Password)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	return nil, fmt.Errorf("Invalid authentication data for %q", tgt.String())
}

func probeHandler(w http.ResponseWriter, r *http.Request, tr *http.Transport) {
	params := r.URL.Query()
	target := params.Get("target")
	if target == "" {
		http.Error(w, "Target parameter missing or empty", http.StatusBadRequest)
		return
	}
	probeSuccessGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Whether or not the probe succeeded",
	})
	probeDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "How many seconds the probe took to complete",
	})
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(*timeoutSeconds)*time.Second)
	defer cancel()
	registry := prometheus.NewRegistry()
	registry.MustRegister(probeSuccessGauge)
	registry.MustRegister(probeDurationGauge)
	start := time.Now()
	success, err := probe(ctx, target, registry, &http.Client{Transport: tr})
	if err != nil {
		log.Printf("Probe request rejected; error is: %v", err)
		http.Error(w, fmt.Sprintf("probe: %v", err), http.StatusBadRequest)
		return
	}
	duration := time.Since(start).Seconds()
	probeDurationGauge.Set(duration)
	if success {
		probeSuccessGauge.Set(1)
		log.Printf("Probe of %q succeeded, took %.3f seconds", target, duration)
	} else {
		// probeSuccessGauge default is 0
		log.Printf("Probe of %q failed, took %.3f seconds", target, duration)
	}
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func main() {
	flag.Parse()

	af, err := ioutil.ReadFile(*authMapFile)
	if err != nil {
		log.Fatalf("Failed to read API authentication map file: %v", err)
	}

	if err := yaml.Unmarshal(af, &authMap); err != nil {
		log.Fatalf("Failed to parse API authentication map file: %v", err)
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("Unable to fetch system CA store: %v", err)
	}
	if *extraCAs != "" {
		certs, err := ioutil.ReadFile(*extraCAs)
		if err != nil {
			log.Fatalf("Failed to read extra CA file: %v", err)
		}

		if ok := roots.AppendCertsFromPEM(certs); !ok {
			log.Fatalf("Failed to append certs from PEM, unknown error")
		}
	}
	tc := &tls.Config{RootCAs: roots}
	if *insecure {
		tc.InsecureSkipVerify = true
	}
	tr := &http.Transport{TLSClientConfig: tc}

	log.Printf("Loaded %d API credentials", len(authMap))

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		probeHandler(w, r, tr)
	})
	go http.ListenAndServe(*listen, nil)
	log.Printf("Spectrum Virtualize exporter running, listening on %q", *listen)
	select {}
}
