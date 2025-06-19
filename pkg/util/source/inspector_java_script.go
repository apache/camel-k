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

package source

import (
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util"
)

// JavaScriptInspector inspects Javascript DSL spec.
type JavaScriptInspector struct {
	baseInspector
}

// Extract extracts all metadata from source spec.
func (i JavaScriptInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	from := util.FindAllDistinctStringSubmatch(
		source.Content,
		singleQuotedFrom,
		doubleQuotedFrom,
		singleQuotedFromF,
		doubleQuotedFromF,
	)
	to := util.FindAllDistinctStringSubmatch(
		source.Content,
		singleQuotedTo,
		doubleQuotedTo,
		singleQuotedToD,
		doubleQuotedToD,
		singleQuotedToF,
		doubleQuotedToF,
		singleQuotedWireTap,
		doubleQuotedWireTap,
	)
	kameletEips := util.FindAllDistinctStringSubmatch(
		source.Content,
		singleQuotedKameletEip,
		doubleQuotedKameletEip)

	hasRest := restRegexp.MatchString(source.Content)

	return i.extract(source, meta, from, to, kameletEips, hasRest)
}

// ReplaceFromURI parses the source content and replace the `from` URI configuration with the a new URI. Returns true if it applies a replacement.
func (i JavaScriptInspector) ReplaceFromURI(source *v1.SourceSpec, newFromURI string) (bool, error) {
	return replaceFromURI(source, newFromURI)
}
