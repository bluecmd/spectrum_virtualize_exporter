// SSH client for Spectrum Virtualize CLI using key or password authentication
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

// The REST server on some systems (seen on a V7000 at 8.4) stops answering
// after running for a while, while the CLI over SSH stays up. The REST API
// is a thin wrapper around the CLI: `rest/lsfoo` returns the rows of
// `lsfoo` as JSON objects keyed by the column headers, with every value a
// string. This client runs `lsfoo -delim ,` over SSH and builds that same
// JSON, so the probes work unchanged on either transport.

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Only list views of info commands: they are the ones with a header row,
// and nothing else should ever be sent to the array.
var sshCommandRe = regexp.MustCompile(`^ls[a-z0-9]+$`)

// One connection per target, kept between scrapes. Logging in is the slow
// part of a probe and the array limits concurrent CLI sessions.
var (
	sshConnsMu sync.Mutex
	sshConns   = map[string]*ssh.Client{}
)

type spectrumSSHClient struct {
	tgt  url.URL
	cfg  *ssh.ClientConfig
	addr string
	ctx  context.Context
}

func newSpectrumSSHClient(ctx context.Context, tgt url.URL, auth Auth, hostKeys ssh.HostKeyCallback) (*spectrumSSHClient, error) {
	var methods []ssh.AuthMethod
	if auth.KeyFile != "" {
		b, err := os.ReadFile(auth.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("reading key file: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(b)
		if err != nil {
			return nil, fmt.Errorf("parsing key file %q: %v", auth.KeyFile, err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if auth.Password != "" {
		methods = append(methods, ssh.Password(auth.Password))
	}
	if auth.User == "" || len(methods) == 0 {
		return nil, fmt.Errorf("Invalid authentication data for %q: need user and keyfile or password", tgt.String())
	}

	addr := tgt.Host
	if tgt.Port() == "" {
		addr = net.JoinHostPort(tgt.Hostname(), "22")
	}
	cfg := &ssh.ClientConfig{
		User:            auth.User,
		Auth:            methods,
		HostKeyCallback: hostKeys,
		Timeout:         10 * time.Second,
	}
	return &spectrumSSHClient{tgt: tgt, cfg: cfg, addr: addr, ctx: ctx}, nil
}

func (c *spectrumSSHClient) conn(fresh bool) (*ssh.Client, error) {
	key := c.tgt.String()
	sshConnsMu.Lock()
	defer sshConnsMu.Unlock()
	if cl, ok := sshConns[key]; ok {
		if !fresh {
			return cl, nil
		}
		cl.Close()
		delete(sshConns, key)
	}
	cl, err := ssh.Dial("tcp", c.addr, c.cfg)
	if err != nil {
		return nil, err
	}
	sshConns[key] = cl
	return cl, nil
}

func (c *spectrumSSHClient) session() (*ssh.Session, error) {
	cl, err := c.conn(false)
	if err != nil {
		return nil, err
	}
	s, err := cl.NewSession()
	if err == nil {
		return s, nil
	}
	// The cached connection went away (array restart, idle timeout); dial
	// once more before giving up.
	cl, err = c.conn(true)
	if err != nil {
		return nil, err
	}
	return cl.NewSession()
}

func (c *spectrumSSHClient) run(cmd string) ([]byte, error) {
	s, err := c.session()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	var stdout, stderr bytes.Buffer
	s.Stdout = &stdout
	s.Stderr = &stderr
	done := make(chan error, 1)
	go func() { done <- s.Run(cmd) }()
	select {
	case <-c.ctx.Done():
		s.Close()
		return nil, c.ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("%q: %v: %s", cmd, err, strings.TrimSpace(stderr.String()))
		}
		return stdout.Bytes(), nil
	}
}

func (c *spectrumSSHClient) Get(path string, query string, obj interface{}) error {
	cmd := strings.TrimPrefix(path, "rest/")
	if !sshCommandRe.MatchString(cmd) || query != "" {
		return fmt.Errorf("unsupported command %q over SSH", path)
	}
	out, err := c.run(cmd + " -delim ,")
	if err != nil {
		return err
	}
	rows, err := parseCLITable(out)
	if err != nil {
		return fmt.Errorf("%s: %v", cmd, err)
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, obj)
}

func (c *spectrumSSHClient) String() string {
	return c.tgt.String()
}

// parseCLITable turns the delimited output of a CLI list view into one map
// per row, keyed by the header. An empty list prints nothing at all.
func parseCLITable(out []byte) ([]map[string]string, error) {
	r := csv.NewReader(bytes.NewReader(out))
	r.FieldsPerRecord = 0 // every row must have as many fields as the header
	r.LazyQuotes = true
	header, err := r.Read()
	if err == io.EOF {
		return []map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	rows := []map[string]string{}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		row := make(map[string]string, len(header))
		for i, h := range header {
			row[h] = rec[i]
		}
		rows = append(rows, row)
	}
}

func sshHostKeyCallback(knownHostsFile string, insecure bool) (ssh.HostKeyCallback, error) {
	if knownHostsFile != "" {
		return knownhosts.New(knownHostsFile)
	}
	if insecure {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	return nil, nil
}
