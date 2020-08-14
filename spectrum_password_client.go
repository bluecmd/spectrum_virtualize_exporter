// HTTP client for Spectrum Virtualize API using user/password authentication
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type spectrumPasswordClient struct {
	tgt url.URL
	hc  HTTPClient
	ctx context.Context
	tok string
}

func (c *spectrumPasswordClient) newPostRequest(url string) (*http.Request, error) {
	r, err := http.NewRequestWithContext(c.ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Add("X-Auth-Token", c.tok)
	return r, nil
}

func (c *spectrumPasswordClient) Get(path string, query string, obj interface{}) error {
	u := c.tgt
	u.Path = path
	u.RawQuery = query

	req, err := c.newPostRequest(u.String())
	if err != nil {
		return err
	}

	req = req.WithContext(c.ctx)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Response code was %d, expected 200", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, obj)
}

func (c *spectrumPasswordClient) String() string {
	return c.tgt.String()
}

func newSpectrumPasswordClient(ctx context.Context, tgt url.URL, hc HTTPClient, user string, passwd string) (*spectrumPasswordClient, error) {
	u := tgt
	u.Path = "/rest/auth"
	r, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Add("X-Auth-Username", user)
	r.Header.Add("X-Auth-Password", passwd)
	resp, err := hc.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Login code was %d, expected 200", resp.StatusCode)
	}

	type login struct {
		Token string
	}
	var obj login

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	return &spectrumPasswordClient{tgt, hc, ctx, obj.Token}, nil
}
