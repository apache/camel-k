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
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/stretchr/testify/require"
)

func assertExtract(t *testing.T, inspector Inspector, source string, assertFn func(meta *Metadata)) {
	t.Helper()

	srcSpec := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Content: source,
		},
	}
	meta := NewMetadata()
	err := inspector.Extract(srcSpec, &meta)
	require.NoError(t, err)

	assertFn(&meta)
}

func assertExtractYAML(t *testing.T, inspector YAMLInspector, source string, assertFn func(meta *Metadata)) {
	t.Helper()

	srcSpec := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "route.yaml",
			Content: source,
		},
		Language: v1.LanguageYaml,
	}
	meta := NewMetadata()
	err := inspector.Extract(srcSpec, &meta)
	require.NoError(t, err)

	assertFn(&meta)
}

func assertExtractYAMLError(t *testing.T, inspector YAMLInspector, source string, assertFn func(err error)) {
	t.Helper()

	srcSpec := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "route.yaml",
			Content: source,
		},
		Language: v1.LanguageYaml,
	}
	meta := NewMetadata()
	err := inspector.Extract(srcSpec, &meta)
	require.Error(t, err)

	assertFn(err)
}
