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
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	camel "github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned"
)

const (
	// NoneRepository is a marker used to indicate that no repository should be used.
	NoneRepository = "none"
)

var DefaultRemoteRepository = NoneRepository

// KameletRepository can be used to obtain a Kamelet definition, looking it up in one or more physical locations.
type KameletRepository interface {

	// List the kamelets available in the repository
	List(ctx context.Context) ([]string, error)

	// Get the Kamelet corresponding to the given name, or nil if not found
	Get(ctx context.Context, name string) (*v1.Kamelet, error)

	// String information about the repository
	String() string
}

// NeNewWithURIsw creates a KameletRepository for the given namespaces and any additional external catalog.
//
// Deprecated: to be removed when dropping support of IntegrationPlatform.
func NewWithURIs(ctx context.Context, client camel.Interface, externalRepos []v1.KameletRepositorySpec, namespaces ...string) (KameletRepository, error) {
	return newRepo(ctx, client, externalRepos, namespaces...)
}

// New creates a KameletRepository for the given namespaces.
func New(ctx context.Context, client camel.Interface, namespaces ...string) (KameletRepository, error) {
	return newRepo(ctx, client, nil, namespaces...)
}

func newRepo(ctx context.Context, client camel.Interface, externalRepos []v1.KameletRepositorySpec, namespaces ...string) (KameletRepository, error) {
	namespaces = makeDistinctNonEmpty(namespaces)
	repoImpls := make([]KameletRepository, 0)
	for _, namespace := range namespaces {
		// Add first a namespace local repository for each namespace
		repoImpls = append(repoImpls, newKubernetesKameletRepository(client, namespace))
	}
	// Deprecated: we will need to remove this part when
	// dropping support for IntegrationPlatform.
	for _, ext := range externalRepos {
		repo, err := newFromURI(ctx, ext.URI)
		if err != nil {
			return nil, err
		}
		repoImpls = append(repoImpls, repo)
	}
	// Add default repo
	defaultRepoImpl, err := newFromURI(ctx, DefaultRemoteRepository)
	if err != nil {
		return nil, err
	}
	repoImpls = append(repoImpls, defaultRepoImpl)

	return newCompositeKameletRepository(repoImpls...), nil
}

func newFromURI(ctx context.Context, uri string) (KameletRepository, error) {
	if uri == NoneRepository {
		return newEmptyKameletRepository(), nil
	} else if after, ok := strings.CutPrefix(uri, "github:"); ok {
		desc := after
		var version string
		if strings.Contains(desc, "@") {
			pos := strings.LastIndex(desc, "@")
			version = desc[pos+1:]
			desc = desc[0:pos]
		}
		parts := strings.Split(desc, "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("expected format is github:owner/repo[/path][@version], got: %s", uri)
		}
		owner := parts[0]
		repo := parts[1]
		var path string

		if len(parts) >= 3 {
			path = strings.Join(parts[2:], "/")
		}

		return newGithubKameletRepository(ctx, owner, repo, path, version), nil
	}

	return nil, fmt.Errorf("invalid uri: %s", uri)
}

func makeDistinctNonEmpty(names []string) []string {
	res := make([]string, 0, len(names))
	presence := make(map[string]bool, len(names))
	for _, n := range names {
		if n == "" || presence[n] {
			continue
		}
		presence[n] = true
		res = append(res, n)
	}

	return res
}
