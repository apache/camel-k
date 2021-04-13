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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

type JavaSourceInspector struct {
	baseInspector
}

func (i JavaSourceInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	from := util.FindAllDistinctStringSubmatch(
		source.Content,
		doubleQuotedFrom,
		doubleQuotedFromF,
	)
	to := util.FindAllDistinctStringSubmatch(
		source.Content,
		doubleQuotedTo,
		doubleQuotedToD,
		doubleQuotedToF,
	)

	meta.FromURIs = append(meta.FromURIs, from...)
	meta.ToURIs = append(meta.ToURIs, to...)

	i.discoverCapabilities(source, meta)
	i.discoverDependencies(source, meta)

	hasRest := restRegexp.MatchString(source.Content)
	if hasRest {
		meta.RequiredCapabilities.Add(v1.CapabilityRest)
	}

	meta.ExposesHTTPServices = hasRest || i.containsHTTPURIs(meta.FromURIs)
	meta.PassiveEndpoints = i.hasOnlyPassiveEndpoints(meta.FromURIs)

	return nil
}
