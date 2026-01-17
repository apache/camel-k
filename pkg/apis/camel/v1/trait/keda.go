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

package trait

// The KEDA trait allows you to configure KEDA autoscalers to scale up and down based of events.
//
// +camel-k:trait=keda.
//
//nolint:godoclint
type KedaTrait struct {
	Trait `json:",inline" property:",squash"`

	// Interval (seconds) to check each trigger on.
	PollingInterval *int32 `json:"pollingInterval,omitempty" property:"polling-interval"`
	// The wait period between the last active trigger reported and scaling the resource back to 0.
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty" property:"cooldown-period"`
	// Enabling this property allows KEDA to scale the resource down to the specified number of replicas.
	IdleReplicaCount *int32 `json:"idleReplicaCount,omitempty" property:"idle-replica-count"`
	// Minimum number of replicas.
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty" property:"min-replica-count"`
	// Maximum number of replicas.
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty" property:"max-replica-count"`
	// Definition of triggers according to the KEDA format. Each trigger must contain `type` field corresponding
	// to the name of a KEDA autoscaler and a key/value map named `metadata` containing specific trigger options
	// and optionally a mapping of secrets, used by Keda operator to poll resources according to the autoscaler type.
	Triggers []KedaTrigger `json:"triggers,omitempty" property:"triggers"`
	// Automatically discover KEDA triggers from Camel component URIs.
	// +kubebuilder:validation:Optional
	Auto *bool `json:"auto,omitempty" property:"auto"`
	// Additional metadata to merge into auto-discovered triggers. Keys are trigger types (e.g., "kafka"),
	// values are maps of metadata key-value pairs to merge (e.g., {"lagThreshold": "10"}).
	// +kubebuilder:validation:Optional
	AutoMetadata map[string]map[string]string `json:"autoMetadata,omitempty" property:"auto-metadata"`
}

type KedaTrigger struct {
	// The autoscaler type.
	Type string `json:"type,omitempty" property:"type"`
	// The trigger metadata (see Keda documentation to learn how to fill for each type).
	Metadata map[string]string `json:"metadata,omitempty" property:"metadata"`
	// The secrets mapping to use. Keda allows the possibility to use values coming from different secrets.
	Secrets []*KedaSecret `json:"secrets,omitempty" property:"secrets"`
}

type KedaSecret struct {
	// The name of the secret to use.
	Name string `json:"name,omitempty" property:"name"`
	// The mapping to use for this secret (eg, `database-secret-key:keda-secret-key`)
	Mapping map[string]string `json:"mapping,omitempty" property:"mapping"`
}
