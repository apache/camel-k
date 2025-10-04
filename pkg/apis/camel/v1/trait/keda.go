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
type KedaTrait struct {
	Trait `property:",squash" json:",inline"`

	// Interval (seconds) to check each trigger on.
	PollingInterval *int32 `property:"polling-interval" json:"pollingInterval,omitempty"`
	// The wait period between the last active trigger reported and scaling the resource back to 0.
	CooldownPeriod *int32 `property:"cooldown-period" json:"cooldownPeriod,omitempty"`
	// Enabling this property allows KEDA to scale the resource down to the specified number of replicas.
	IdleReplicaCount *int32 `property:"idle-replica-count" json:"idleReplicaCount,omitempty"`
	// Minimum number of replicas.
	MinReplicaCount *int32 `property:"min-replica-count" json:"minReplicaCount,omitempty"`
	// Maximum number of replicas.
	MaxReplicaCount *int32 `property:"max-replica-count" json:"maxReplicaCount,omitempty"`
	// Definition of triggers according to the KEDA format. Each trigger must contain `type` field corresponding
	// to the name of a KEDA autoscaler and a key/value map named `metadata` containing specific trigger options
	// and optionally a mapping of secrets, used by Keda operator to poll resources according to the autoscaler type.
	Triggers []KedaTrigger `property:"triggers" json:"triggers,omitempty"`
}

type KedaTrigger struct {
	// The autoscaler type.
	Type string `property:"type" json:"type,omitempty"`
	// The trigger metadata (see Keda documentation to learn how to fill for each type).
	Metadata map[string]string `property:"metadata" json:"metadata,omitempty"`
	// The secrets mapping to use. Keda allows the possibility to use values coming from different secrets.
	Secrets []*KedaSecret `property:"secrets" json:"secrets,omitempty"`
}

type KedaSecret struct {
	// The name of the secret to use.
	Name string `property:"name" json:"name,omitempty"`
	// The mapping to use for this secret (eg, `database-secret-key:keda-secret-key`)
	Mapping map[string]string `property:"mapping" json:"mapping,omitempty"`
}
