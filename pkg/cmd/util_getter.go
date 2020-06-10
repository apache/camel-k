/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
)

var Getters map[string]Getter

func init() {
	Getters = map[string]Getter{
		"http":   HTTPGetter{},
		"https":  HTTPGetter{},
		"github": GitHubGetter{},
	}
}

type Getter interface {
	Get(u *url.URL) ([]byte, error)
}

// A simple getter that retrieves the content of an integration from an
// http(s) endpoint.
type HTTPGetter struct {
}

func (g HTTPGetter) Get(u *url.URL) ([]byte, error) {
	return g.doGet(u.String())
}

func (g HTTPGetter) doGet(source string) ([]byte, error) {
	// nolint: gosec
	resp, err := http.Get(source)
	if err != nil {
		return []byte{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return []byte{}, fmt.Errorf("the provided URL %s is not reachable, error code is %d", source, resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return content, nil
}

// A simple getter that retrieves the content of an integration from
// a GitHub endpoint using a RAW endpoint.
type GitHubGetter struct {
	HTTPGetter
}

func (g GitHubGetter) Get(u *url.URL) ([]byte, error) {
	src := u.Scheme + ":" + u.Opaque
	re := regexp.MustCompile(`^github:([^/]+)/([^/]+)/(.+)$`)

	items := re.FindStringSubmatch(src)
	if len(items) != 4 {
		return []byte{}, fmt.Errorf("malformed github url: %s", src)
	}

	branch := u.Query().Get("branch")
	if branch == "" {
		branch = "master"
	}

	srcURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", items[1], items[2], branch, items[3])

	return g.HTTPGetter.doGet(srcURL)
}
