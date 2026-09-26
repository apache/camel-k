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

package apis

import (
	corev1 "k8s.io/api/core/v1"
)

// Conditions is the schema for the conditions portion of the payload.
type Conditions []Condition

// ConditionType is a camel-cased condition type.
type ConditionType string

// ConditionReady specifies that the resource is ready.
const ConditionReady ConditionType = "Ready"

// ConditionSeverity expresses the severity of a Condition Type failing.
type ConditionSeverity string

// Condition defines a readiness condition for a Knative resource.
type Condition struct {
	// Type of condition.
	Type ConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`

	// Severity with which to treat failures of this type of condition.
	// +optional
	Severity ConditionSeverity `json:"severity,omitempty"`

	// LastTransitionTime is the last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime VolatileTime `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// IsTrue is true if the condition is True.
func (c *Condition) IsTrue() bool {
	if c == nil {
		return false
	}

	return c.Status == corev1.ConditionTrue
}

// IsFalse is true if the condition is False.
func (c *Condition) IsFalse() bool {
	if c == nil {
		return false
	}

	return c.Status == corev1.ConditionFalse
}

// GetReason returns the reason of the condition, or an empty string for a nil condition.
func (c *Condition) GetReason() string {
	if c == nil {
		return ""
	}

	return c.Reason
}

// GetMessage returns the message of the condition, or an empty string for a nil condition.
func (c *Condition) GetMessage() string {
	if c == nil {
		return ""
	}

	return c.Message
}
