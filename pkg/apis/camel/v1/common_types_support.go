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

package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (in *Artifact) String() string {
	return in.ID
}

func (in *ConfigurationSpec) String() string {
	return fmt.Sprintf("%s=%s", in.Type, in.Value)
}

func (in *RuntimeSpec) CapabilityDependencies(capability string) []MavenArtifact {
	deps := make([]MavenArtifact, 0)

	if capability, ok := in.Capabilities[capability]; ok {
		deps = append(deps, capability.Dependencies...)
	}

	return deps
}

// TraitProfileByName returns the trait profile corresponding to the given name (case insensitive).
func TraitProfileByName(name string) TraitProfile {
	for _, p := range AllTraitProfiles {
		if strings.EqualFold(name, string(p)) {
			return p
		}
	}
	return ""
}

// Equal checks if the profile is equal to the given profile (case insensitive).
func (p TraitProfile) Equal(other TraitProfile) bool {
	return strings.EqualFold(string(p), string(other))
}

// MarshalJSON returns m as the JSON encoding of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

var _ json.Marshaler = (*RawMessage)(nil)
var _ json.Unmarshaler = (*RawMessage)(nil)
