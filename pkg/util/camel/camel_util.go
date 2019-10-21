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
	"sort"

	"github.com/Masterminds/semver"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/log"
)

func findBestMatch(catalogs []v1alpha1.CamelCatalog, camelVersion string, runtimeVersion string) (*RuntimeCatalog, error) {
	for _, catalog := range catalogs {
		if catalog.Spec.Version == camelVersion && catalog.Spec.RuntimeVersion == runtimeVersion {
			return NewRuntimeCatalog(catalog.Spec), nil
		}
	}

	vc := newSemVerConstraint(camelVersion)
	rc := newSemVerConstraint(runtimeVersion)
	if vc == nil || rc == nil {
		return nil, nil
	}

	cc := newCatalogVersionCollection(catalogs)
	for _, x := range cc {
		if vc.Check(x.Version) && rc.Check(x.RuntimeVersion) {
			return NewRuntimeCatalog(x.Catalog.Spec), nil
		}
	}

	return nil, nil
}

func newSemVerConstraint(versionConstraint string) *semver.Constraints {
	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil || constraint == nil {
		if err != nil {
			log.Debug("Unable to parse version constraint: %s, error:\n", versionConstraint, err.Error())
		}
		if constraint == nil {
			log.Debug("Unable to parse version constraint: %s\n", versionConstraint)
		}
	}

	return constraint
}

func newCatalogVersionCollection(catalogs []v1alpha1.CamelCatalog) CatalogVersionCollection {
	versions := make([]CatalogVersion, 0, len(catalogs))

	for i := range catalogs {
		cv, err := semver.NewVersion(catalogs[i].Spec.Version)
		if err != nil {
			log.Debugf("Invalid semver version (camel) %s", cv)
			continue
		}

		rv, err := semver.NewVersion(catalogs[i].Spec.RuntimeVersion)
		if err != nil {
			log.Debugf("Invalid semver version (runtime) %s", rv)
			continue
		}

		versions = append(versions, CatalogVersion{
			Version:        cv,
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
