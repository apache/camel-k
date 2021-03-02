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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	camel "github.com/apache/camel-k/pkg/client/camel/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NoneRepository is a marker used to indicate that no repository should be used
	NoneRepository = "none"
)

var DefaultRemoteRepository = NoneRepository

// KameletRepository can be used to obtain a Kamelet definition, looking it up in one or more physical locations
type KameletRepository interface {

	// List the kamelets available in the repository
	List(ctx context.Context) ([]string, error)

	// Get the Kamelet corresponding to the given name, or nil if not found
	Get(ctx context.Context, name string) (*v1alpha1.Kamelet, error)

	// String information about the repository
	String() string
}

// New creates a KameletRepository for the given namespaces.
// Kamelets are first looked up in all the given namespaces, in the order they appear.
// If one namespace defines an IntegrationPlatform (only the first IntegrationPlatform in state "Ready" found),
// then all kamelet repository URIs defined in the IntegrationPlatform are included.
func New(ctx context.Context, client camel.Interface, namespaces ...string) (KameletRepository, error) {
	namespaces = makeDistinctNonEmpty(namespaces)
	platform, err := lookupPlatform(ctx, client, namespaces...)
	if err != nil {
		return nil, err
	}
	return NewForPlatform(ctx, client, platform, namespaces...)
}

// NewForPlatform creates a KameletRepository for the given namespaces and platform.
// Kamelets are first looked up in all the given namespaces, in the order they appear,
// then repositories defined in the platform are looked up.
func NewForPlatform(ctx context.Context, client camel.Interface, platform *v1.IntegrationPlatform, namespaces ...string) (KameletRepository, error) {
	namespaces = makeDistinctNonEmpty(namespaces)
	repoImpls := make([]KameletRepository, 0)
	for _, namespace := range namespaces {
		// Add first a namespace local repository for each namespace
		repoImpls = append(repoImpls, newKubernetesKameletRepository(client, namespace))
	}
	if platform != nil {
		repos := getRepositoriesFromPlatform(platform)
		for _, repoURI := range repos {
			repoImpl, err := newFromURI(repoURI)
			if err != nil {
				return nil, err
			}
			repoImpls = append(repoImpls, repoImpl)
		}
	} else {
		// Add default repo
		defaultRepoImpl, err := newFromURI(DefaultRemoteRepository)
		if err != nil {
			return nil, err
		}
		repoImpls = append(repoImpls, defaultRepoImpl)
	}

	return newCompositeKameletRepository(repoImpls...), nil
}

// NewStandalone creates a KameletRepository that can be used in cases where there's no connection to a Kubernetes cluster.
// The given uris are used to construct the repositories.
// If the uris parameter is nil, then only the DefaultRemoteRepository will be included.
func NewStandalone(uris ...string) (KameletRepository, error) {
	repoImpls := make([]KameletRepository, 0, len(uris)+1)
	for _, repoURI := range uris {
		repoImpl, err := newFromURI(repoURI)
		if err != nil {
			return nil, err
		}
		if repoImpl != nil {
			repoImpls = append(repoImpls, repoImpl)
		}
	}
	if len(repoImpls) == 0 {
		defaultRepoImpl, err := newFromURI(DefaultRemoteRepository)
		if err != nil {
			return nil, err
		}
		if defaultRepoImpl != nil {
			repoImpls = append(repoImpls, defaultRepoImpl)
		}
	}
	return newCompositeKameletRepository(repoImpls...), nil
}

func lookupPlatform(ctx context.Context, client camel.Interface, namespaces ...string) (*v1.IntegrationPlatform, error) {
	for _, namespace := range namespaces {
		pls, err := client.CamelV1().IntegrationPlatforms(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pl := range pls.Items {
			if pl.Status.Phase == v1.IntegrationPlatformPhaseReady {
				return &pl, nil
			}
		}
		if len(pls.Items) > 0 {
			// If none is ready, return the first one
			return &pls.Items[0], nil
		}
	}
	return nil, nil
}

func getRepositoriesFromPlatform(platform *v1.IntegrationPlatform) []string {
	if platform == nil {
		return nil
	}
	repos := platform.Status.Kamelet.Repositories
	if len(repos) == 0 {
		// Maybe not reconciled yet
		repos = platform.Spec.Kamelet.Repositories
	}
	res := make([]string, 0, len(repos))
	for _, repo := range repos {
		res = append(res, repo.URI)
	}
	return res
}

func newFromURI(uri string) (KameletRepository, error) {
	if uri == NoneRepository {
		return newEmptyKameletRepository(), nil
	} else if strings.HasPrefix(uri, "github:") {
		desc := strings.TrimPrefix(uri, "github:")
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
		return newGithubKameletRepository(owner, repo, path, version), nil
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
