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
	"strings"

	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/gzip"
)

const (
	// Megabyte represent the related unit.
	Megabyte = 1 << 20
	// Kilobyte represent the related unit.
	Kilobyte = 1 << 10
)

func LoadRawContent(ctx context.Context, source string) ([]byte, string, error) {
	var content []byte
	var err error

	ok, err := IsLocalAndFileExists(source)
	if err != nil {
		return nil, "", err
	}

	if ok {
		content, err = util.ReadFile(source)
	} else {
		var u *url.URL
		u, err = url.Parse(source)
		if err != nil {
			return nil, "", err
		}

		switch u.Scheme {
		case "github":
			content, err = loadContentGitHub(ctx, u)
		case "http":
			content, err = loadContentHTTP(ctx, u)
		case "https":
			content, err = loadContentHTTP(ctx, u)
		default:
			return nil, "", fmt.Errorf("missing file or unsupported scheme %s", u.Scheme)
		}
	}

	if err != nil {
		return nil, "", err
	}

	contentType := http.DetectContentType(content)
	return content, contentType, nil
}

func IsBinary(contentType string) bool {
	// According the http.DetectContentType method
	// also json and other "text" application mime types would be reported as text
	return !strings.HasPrefix(contentType, "text")
}

func CompressToString(content []byte) (string, error) {
	bytes, err := gzip.CompressBase64(content)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func LoadTextContent(ctx context.Context, source string, base64Compression bool) (string, string, bool, error) {
	content, contentType, err := LoadRawContent(ctx, source)
	if err != nil {
		return "", "", false, err
	}

	if base64Compression {
		base64Compressed, err := CompressToString(content)
		return base64Compressed, contentType, true, err
	}

	return string(content), contentType, false, nil
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

	if resp.StatusCode != 200 {
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
