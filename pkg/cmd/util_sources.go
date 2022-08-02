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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/apache/camel-k/pkg/util"
	"github.com/spf13/cobra"

	"golang.org/x/oauth2"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
)

// Source represents the source file of an Integration.
type Source struct {
	Origin   string
	Location string
	Name     string
	Content  string
	Compress bool
	Local    bool
}

func (s *Source) setContent(content []byte) error {
	if s.Compress {
		result, err := compressToString(content)
		if err != nil {
			return err
		}

		s.Content = result
	} else {
		s.Content = string(content)
	}

	return nil
}

// ResolveSources resolves sources from a variety of locations including local and remote.
func ResolveSources(ctx context.Context, locations []string, compress bool, cmd *cobra.Command) ([]Source, error) {
	sources := make([]Source, 0, len(locations))

	for _, location := range locations {
		ok, err := isLocalAndFileExists(location)
		if err != nil {
			return sources, err
		}

		if ok {
			answer, err := resolveLocalSource(location, compress)
			if err != nil {
				return sources, err
			}

			sources = append(sources, answer)
		} else {
			u, err := url.Parse(location)
			if err != nil {
				return sources, err
			}

			switch {
			case u.Scheme == gistScheme || strings.HasPrefix(location, "https://gist.github.com/"):
				answer, err := resolveGistSource(ctx, location, compress, cmd, u)
				if err != nil {
					return sources, err
				}
				sources = append(sources, answer...)
			case u.Scheme == githubScheme:
				answer, err := resolveSource(location, compress, func() ([]byte, error) {
					return loadContentGitHub(ctx, u)
				})
				if err != nil {
					return sources, err
				}
				sources = append(sources, answer)
			case u.Scheme == httpScheme || u.Scheme == httpsScheme:
				answer, err := resolveSource(location, compress, func() ([]byte, error) {
					return loadContentHTTP(ctx, u)
				})
				if err != nil {
					return sources, err
				}
				sources = append(sources, answer)
			default:
				return sources, fmt.Errorf("missing file or unsupported scheme in %s", location)
			}
		}
	}

	return sources, nil
}

// resolveGistSource resolves sources from a Gist.
func resolveGistSource(ctx context.Context, location string, compress bool, cmd *cobra.Command, u *url.URL) ([]Source, error) {
	var hc *http.Client

	if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		hc = oauth2.NewClient(ctx, ts)

		fmt.Fprintln(cmd.OutOrStdout(), "GITHUB_TOKEN env var detected, using it for GitHub APIs authentication")
	}

	gc := github.NewClient(hc)
	gistID := ""

	if strings.HasPrefix(location, "https://gist.github.com/") {
		names := util.FindNamedMatches(`^https://gist.github.com/(([a-zA-Z0-9]*)/)?(?P<gistid>[a-zA-Z0-9]*)$`, location)
		if value, ok := names["gistid"]; ok {
			gistID = value
		}
	} else {
		gistID = u.Opaque
	}

	if gistID == "" {
		return []Source{}, fmt.Errorf("unable to determining gist id from %s", location)
	}

	gists, _, err := gc.Gists.Get(ctx, gistID)
	if err != nil {
		return []Source{}, err
	}

	sources := make([]Source, 0, len(gists.Files))
	for _, v := range gists.Files {
		if v.Filename == nil || v.Content == nil {
			continue
		}

		answer := Source{
			Name:     *v.Filename,
			Compress: compress,
			Origin:   location,
		}
		if v.RawURL != nil {
			answer.Location = *v.RawURL
		}
		if err := answer.setContent([]byte(*v.Content)); err != nil {
			return sources, err
		}
		sources = append(sources, answer)
	}

	return sources, nil
}

// resolveLocalSource resolves a source from the local file system.
func resolveLocalSource(location string, compress bool) (Source, error) {
	if _, err := os.Stat(location); err != nil && os.IsNotExist(err) {
		return Source{}, errors.Wrapf(err, "file %s does not exist", location)
	} else if err != nil {
		return Source{}, errors.Wrapf(err, "error while accessing file %s", location)
	}

	answer, err := resolveSource(location, compress, func() ([]byte, error) {
		return util.ReadFile(location)
	})
	if err != nil {
		return Source{}, err
	}
	answer.Local = true

	return answer, nil
}

// resolveSource resolves a source using the content provider function.
func resolveSource(location string, compress bool, loadContent func() ([]byte, error)) (Source, error) {
	// strip query part from location if any
	locPath := util.SubstringBefore(location, "?")
	if locPath == "" {
		locPath = location
	}
	answer := Source{
		Name:     path.Base(locPath),
		Origin:   location,
		Location: location,
		Compress: compress,
	}

	content, err := loadContent()
	if err != nil {
		return Source{}, err
	}
	if err := answer.setContent(content); err != nil {
		return Source{}, err
	}

	return answer, nil
}
