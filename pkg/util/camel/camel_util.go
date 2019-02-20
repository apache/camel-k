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

// FindBestMatch --
func FindBestMatch(version string, catalogs []v1alpha1.CamelCatalog) (*RuntimeCatalog, error) {
	constraint, err := semver.NewConstraint(version)

	//
	// if the version is not a constraint, use exact match
	//
	if err != nil || constraint == nil {
		if err != nil {
			log.Debug("Unable to parse constraint: %s, error:\n", version, err.Error())
		}
		if constraint == nil {
			log.Debug("Unable to parse constraint: %s\n", version)
		}

		return FindExactMatch(version, catalogs)
	}

	return FindBestSemVerMatch(constraint, catalogs)
}

// FindExactMatch --
func FindExactMatch(version string, catalogs []v1alpha1.CamelCatalog) (*RuntimeCatalog, error) {
	for _, catalog := range catalogs {
		if catalog.Spec.Version == version {
			return NewRuntimeCatalog(catalog.Spec), nil
		}
	}

	return nil, nil
}

// FindBestSemVerMatch --
func FindBestSemVerMatch(constraint *semver.Constraints, catalogs []v1alpha1.CamelCatalog) (*RuntimeCatalog, error) {
	versions := make([]*semver.Version, 0)

	for _, catalog := range catalogs {
		v, err := semver.NewVersion(catalog.Spec.Version)
		if err != nil {
			log.Debugf("Invalid semver version %s, skip it", catalog.Spec.Version)
		}

		versions = append(versions, v)
	}

	sort.Sort(
		sort.Reverse(semver.Collection(versions)),
	)

	for _, v := range versions {
		ver := v

		if constraint.Check(ver) {
			for _, catalog := range catalogs {
				if catalog.Spec.Version == ver.Original() {
					return NewRuntimeCatalog(catalog.Spec), nil
				}
			}

		}
	}

	return nil, nil
}
