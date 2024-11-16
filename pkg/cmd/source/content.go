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

package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/apache/camel-k/v2/pkg/util/gzip"
)

const (
	// Megabyte represent the related unit.
	Megabyte = 1 << 20
	// Kilobyte represent the related unit.
	Kilobyte = 1 << 10
)

func CompressToString(content []byte) (string, error) {
	bytes, err := gzip.CompressBase64(content)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func loadContentHTTP(ctx context.Context, u fmt.Stringer) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	c := &http.Client{}

	resp, err := c.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("the provided URL %s is not reachable, error code is %d", u.String(), resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func loadContentGitHub(ctx context.Context, u *url.URL) ([]byte, error) {
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

	return loadContentHTTP(ctx, rawURL)
}
