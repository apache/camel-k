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

package metadata

import (
	"sort"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
)

// discoverDependencies returns a list of dependencies required by the given source code
func discoverDependencies(source v1alpha1.SourceSpec, fromURIs []string, toURIs []string) []string {
	candidateMap := make(map[string]bool)
	uris := make([]string, 0, len(fromURIs)+len(toURIs))
	uris = append(uris, fromURIs...)
	uris = append(uris, toURIs...)
	for _, uri := range uris {
		candidateComp := decodeComponent(uri)
		if candidateComp != "" {
			candidateMap[candidateComp] = true
		}
	}
	// Remove duplicates and sort
	candidateComponents := make([]string, 0, len(candidateMap))
	for cmp := range candidateMap {
		candidateComponents = append(candidateComponents, cmp)
	}
	sort.Strings(candidateComponents)
	return candidateComponents
}

func decodeComponent(uri string) string {
	uriSplit := strings.SplitN(uri, ":", 2)
	if len(uriSplit) < 2 {
		return ""
	}
	uriStart := uriSplit[0]
	if component := camel.Runtime.GetArtifactByScheme(uriStart); component != nil {
		artifactID := component.ArtifactID
		if strings.HasPrefix(artifactID, "camel-") {
			return "camel:" + artifactID[6:]
		}
		return "mvn:" + component.GroupID + ":" + artifactID + ":" + component.Version
	}
	return ""
}
