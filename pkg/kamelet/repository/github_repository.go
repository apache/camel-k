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

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"sort"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type githubKameletRepository struct {
	httpClient *http.Client
	owner      string
	repo       string
	path       string
	ref        string
}

func newGithubKameletRepository(ctx context.Context, owner, repo, path, ref string) KameletRepository {
	httpClient := &http.Client{}
	if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		ctx := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
		httpClient = oauth2.NewClient(ctx, ts)
	}

	return &githubKameletRepository{
		httpClient: httpClient,
		owner:      owner,
		repo:       repo,
		path:       path,
		ref:        ref,
	}
}

// Enforce type.
var _ KameletRepository = &githubKameletRepository{}

func (c *githubKameletRepository) List(ctx context.Context) ([]string, error) {
	dir, err := c.listFiles(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(dir))
	for _, file := range dir {
		if file != nil && file.Name != nil && isKameletFileName(*file.Name) {
			res = append(res, getKameletNameFromFile(*file.Name))
		}
	}
	sort.Strings(res)
	return res, nil
}

func (c *githubKameletRepository) Get(ctx context.Context, name string) (*v1.Kamelet, error) {
	dir, err := c.listFiles(ctx)
	if err != nil {
		return nil, err
	}

	for _, file := range dir {
		if file == nil || file.Name == nil {
			continue
		}
		if isFileNameForKamelet(name, *file.Name) && file.DownloadURL != nil {
			kamelet, err := c.downloadKamelet(ctx, *file.DownloadURL)
			if err != nil {
				return kamelet, err
			}
			if kamelet.Name != name {
				return nil, fmt.Errorf("kamelet names do not match: expected %s, got %s", name, kamelet.Name)
			}
			return kamelet, nil
		}
	}
	return nil, nil
}

func (c *githubKameletRepository) listFiles(ctx context.Context) ([]*github.RepositoryContent, error) {
	gc := github.NewClient(c.httpClient)
	var ref *github.RepositoryContentGetOptions
	if c.ref != "" {
		ref = &github.RepositoryContentGetOptions{Ref: c.ref}
	}
	_, dir, _, err := gc.Repositories.GetContents(ctx, c.owner, c.repo, c.path, ref)
	return dir, err
}

func (c *githubKameletRepository) downloadKamelet(ctx context.Context, url string) (*v1.Kamelet, error) {
	return downloadGithubKamelet(ctx, url, c.httpClient)
}

func downloadGithubKamelet(ctx context.Context, url string, httpClient *http.Client) (*v1.Kamelet, error) {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot download file %s: %d %s", url, resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(parsedURL.Path, ".yaml") || strings.HasSuffix(parsedURL.Path, ".yml") {
		content, err = yaml.ToJSON(content)
		if err != nil {
			return nil, err
		}
	}

	var kamelet v1.Kamelet
	if err := json.Unmarshal(content, &kamelet); err != nil {
		return nil, err
	}
	return &kamelet, nil
}

func (c *githubKameletRepository) String() string {
	return fmt.Sprintf("Github[owner=%s, repo=%s, path=%s, ref=%s]", c.owner, c.repo, c.path, c.ref)
}
