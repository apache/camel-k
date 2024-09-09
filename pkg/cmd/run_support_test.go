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

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFileLocation(t *testing.T) {
	optionFileLocations := []string{
		"file:/path/to/valid/file",
		"file:app.properties",
		"configmap:my-configmap",
		"secret:my-secret",
		"file:/validfile@/tmp/destination",
	}

	filteredOptions := filterFileLocation(optionFileLocations)

	assert.Equal(t, 3, len(filteredOptions))
	assert.Equal(t, "/path/to/valid/file", filteredOptions[0])
	assert.Equal(t, "app.properties", filteredOptions[1])
	assert.Equal(t, "/validfile", filteredOptions[2])
}

func TestExtractTraitNames(t *testing.T) {
	traitProps := []string{"container.enabled=true", "no-trait.noval=1", "nothing"}
	tn := extractTraitNames(traitProps)
	assert.Equal(t, "container", tn[0])
	assert.Equal(t, "no-trait", tn[1])
	assert.Equal(t, "nothing", tn[2])
}
