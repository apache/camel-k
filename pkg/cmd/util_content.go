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
	"strings"
)

func loadContent(source string, compress bool, compressBinary bool) (string, bool, error) {
	var content []byte
	var err error

	if isLocalAndFileExists(source) {
		content, err = ioutil.ReadFile(source)
	} else {
		u, err := url.Parse(source)
		if err != nil {
			return "", false, err
		}

		switch u.Scheme {
		case "github":
			content, err = loadContentGitHub(u)
		case "http":
			content, err = loadContentHTTP(u)
		case "https":
			content, err = loadContentHTTP(u)
		default:
			return "", false, fmt.Errorf("unsupported scheme %s", u.Scheme)
		}
	}

	if err != nil {
		return "", false, err
	}
	doCompress := compress
	if !doCompress && compressBinary {
		contentType := http.DetectContentType(content)
		if strings.HasPrefix(contentType, "application/octet-stream") {
			doCompress = true
		}
	}

	if doCompress {
		answer, err := compressToString(content)
		return answer, true, err
	}

	return string(content), false, nil
}

func loadContentHTTP(u *url.URL) ([]byte, error) {
	// nolint: gosec
	resp, err := http.Get(u.String())
	if err != nil {
		return []byte{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return []byte{}, fmt.Errorf("the provided URL %s is not reachable, error code is %d", u.String(), resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return content, nil
}

func loadContentGitHub(u *url.URL) ([]byte, error) {
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
	rawURL, err := url.Parse(srcURL)
	if err != nil {
		return []byte{}, err
	}

	return loadContentHTTP(rawURL)
}
