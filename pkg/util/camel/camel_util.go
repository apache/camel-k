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
	"fmt"
	"sort"

	"github.com/Masterminds/semver"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// FindBestMatch --
func FindBestMatch(constraint string, catalogs []v1alpha1.CamelCatalog) (*RuntimeCatalog, error) {
	ref, err := semver.NewConstraint(constraint)
	if err != nil {
		fmt.Printf("Error parsing version: %s\n", err.Error())
	}

	versions := make([]*semver.Version, 0)

	for _, catalog := range catalogs {
		v, err := semver.NewVersion(catalog.Spec.Version)
		if err != nil {
			return nil, err
		}

		versions = append(versions, v)
	}

	sort.Sort(
		sort.Reverse(semver.Collection(versions)),
	)

	for _, v := range versions {
		ver := v

		if ref.Check(ver) {
			for _, catalog := range catalogs {
				if catalog.Spec.Version == ver.Original() {
					return NewRuntimeCatalog(catalog.Spec), nil
				}
			}

		}
	}

	return nil, nil

}
