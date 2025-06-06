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

package camel

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

var (
	BasePath         = "/etc/camel"
	ConfDPath        = filepath.Join(BasePath, "conf.d")
	ResourcesDPath   = filepath.Join(BasePath, "resources.d")
	SourcesMountPath = filepath.Join(BasePath, "sources")
	// Deprecated: replaced by /etc/camel/resources.d/[_configmaps/_secrets] (ResourcesConfigmapsMountPath/ResourcesSecretsMountPath).
	ResourcesDefaultMountPath    = filepath.Join(BasePath, "resources")
	ResourcesConfigmapsMountPath = filepath.Join(ResourcesDPath, "_configmaps")
	ResourcesSecretsMountPath    = filepath.Join(ResourcesDPath, "_secrets")
	ConfigConfigmapsMountPath    = filepath.Join(ConfDPath, "_configmaps")
	ConfigSecretsMountPath       = filepath.Join(ConfDPath, "_secrets")
	ServiceBindingsMountPath     = filepath.Join(ConfDPath, "_servicebindings")
)

func findBestMatch(catalogs []v1.CamelCatalog, runtime v1.RuntimeSpec) (*RuntimeCatalog, error) {
	for _, catalog := range catalogs {
		if catalog.Spec.Runtime.Version == runtime.Version && catalog.Spec.Runtime.Provider == runtime.Provider {
			return NewRuntimeCatalog(catalog), nil
		}
	}

	rc := newSemVerConstraint(runtime.Version)
	if rc == nil {
		return nil, nil
	}

	cc := newCatalogVersionCollection(catalogs)
	for _, c := range cc {
		if rc.Check(c.RuntimeVersion) {
			return NewRuntimeCatalog(*c.Catalog), nil
		}
	}

	return nil, nil
}

func newSemVerConstraint(versionConstraint string) *semver.Constraints {
	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil || constraint == nil {
		if err != nil {
			log.Debugf("Unable to parse version constraint: %s, error: %s", versionConstraint, err.Error())
		}
		if constraint == nil {
			log.Debugf("Unable to parse version constraint: %s", versionConstraint)
		}
	}

	return constraint
}

func newCatalogVersionCollection(catalogs []v1.CamelCatalog) CatalogVersionCollection {
	versions := make([]CatalogVersion, 0, len(catalogs))

	for i := range catalogs {
		rv, err := semver.NewVersion(catalogs[i].Spec.Runtime.Version)
		if err != nil {
			log.Debugf("Invalid semver version (runtime) %s", rv)

			continue
		}

		versions = append(versions, CatalogVersion{
			RuntimeVersion: rv,
			Catalog:        &catalogs[i],
		})
	}

	answer := CatalogVersionCollection(versions)

	sort.Sort(
		sort.Reverse(answer),
	)

	return answer
}

func getDependency(artifact v1.CamelArtifact, runtimeProvider v1.RuntimeProvider) string {
	if runtimeProvider.IsQuarkusBased() {
		return strings.Replace(artifact.ArtifactID, "camel-quarkus-", "camel:", 1)
	}
	return strings.Replace(artifact.ArtifactID, "camel-", "camel:", 1)
}
