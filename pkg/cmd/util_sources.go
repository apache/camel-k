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

	"golang.org/x/oauth2"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
)

// Source ---
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

// ResolveSources ---
func ResolveSources(ctx context.Context, locations []string, compress bool) ([]Source, error) {
	sources := make([]Source, 0, len(locations))

	for _, location := range locations {
		ok, err := isLocalAndFileExists(location)
		if err != nil {
			return sources, err
		}

		if ok {
			answer, err := ResolveLocalSource(location, compress)
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
				var tc *http.Client

				if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
					ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
					tc = oauth2.NewClient(ctx, ts)

					fmt.Println("GITHUB_TOKEN env var detected, using it for GitHub APIs authentication")
				}

				gc := github.NewClient(tc)
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
					return sources, fmt.Errorf("unable to determining gist id from %s", location)
				}

				gists, _, err := gc.Gists.Get(ctx, gistID)
				if err != nil {
					return sources, err
				}

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
			case u.Scheme == githubScheme:
				answer := Source{
					Name:     path.Base(location),
					Origin:   location,
					Location: location,
					Compress: compress,
				}

				content, err := loadContentGitHub(ctx, u)
				if err != nil {
					return sources, err
				}
				if err := answer.setContent(content); err != nil {
					return sources, err
				}
				sources = append(sources, answer)
			case u.Scheme == httpScheme:
				answer := Source{
					Name:     path.Base(location),
					Origin:   location,
					Location: location,
					Compress: compress,
				}

				content, err := loadContentHTTP(ctx, u)
				if err != nil {
					return sources, err
				}
				if err := answer.setContent(content); err != nil {
					return sources, err
				}
				sources = append(sources, answer)
			case u.Scheme == httpsScheme:
				answer := Source{
					Name:     path.Base(location),
					Origin:   location,
					Location: location,
					Compress: compress,
				}

				content, err := loadContentHTTP(ctx, u)
				if err != nil {
					return sources, err
				}
				if err := answer.setContent(content); err != nil {
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

// ResolveLocalSource --
func ResolveLocalSource(location string, compress bool) (Source, error) {
	if _, err := os.Stat(location); err != nil && os.IsNotExist(err) {
		return Source{}, errors.Wrapf(err, "file %s does not exist", location)
	} else if err != nil {
		return Source{}, errors.Wrapf(err, "error while accessing file %s", location)
	}

	answer := Source{
		Name:     path.Base(location),
		Origin:   location,
		Location: location,
		Compress: compress,
		Local:    true,
	}

	content, err := util.ReadFile(location)
	if err != nil {
		return Source{}, err
	}
	if err := answer.setContent(content); err != nil {
		return Source{}, err
	}

	return answer, nil
}
