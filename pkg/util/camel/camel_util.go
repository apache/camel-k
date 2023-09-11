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
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

var (
	BasePath                  = "/etc/camel"
	ConfDPath                 = filepath.Join(BasePath, "conf.d")
	SourcesMountPath          = filepath.Join(BasePath, "sources")
	ResourcesDefaultMountPath = filepath.Join(BasePath, "resources")
	ConfigResourcesMountPath  = filepath.Join(ConfDPath, "_resources")
	ConfigConfigmapsMountPath = filepath.Join(ConfDPath, "_configmaps")
	ConfigSecretsMountPath    = filepath.Join(ConfDPath, "_secrets")
	ServiceBindingsMountPath  = filepath.Join(ConfDPath, "_servicebindings")
)

func findCatalog(catalogs []v1.CamelCatalog, runtime v1.RuntimeSpec) (*RuntimeCatalog, error) {
	for _, catalog := range catalogs {
		if catalog.Spec.Runtime.Version == runtime.Version && catalog.Spec.Runtime.Provider == runtime.Provider {
			return NewRuntimeCatalog(catalog), nil
		}
	}
	return nil, nil
}

func getDependency(artifact v1.CamelArtifact, runtimeProvider v1.RuntimeProvider) string {
	if runtimeProvider == v1.RuntimeProviderQuarkus {
		return strings.Replace(artifact.ArtifactID, "camel-quarkus-", "camel:", 1)
	}
	return strings.Replace(artifact.ArtifactID, "camel-", "camel:", 1)
}
