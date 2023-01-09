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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildStatusApplyConfiguration represents an declarative configuration of the BuildStatus type for use
// with apply.
type BuildStatusApplyConfiguration struct {
	ObservedGeneration *int64                             `json:"observedGeneration,omitempty"`
	Phase              *v1.BuildPhase                     `json:"phase,omitempty"`
	Image              *string                            `json:"image,omitempty"`
	Digest             *string                            `json:"digest,omitempty"`
	BaseImage          *string                            `json:"baseImage,omitempty"`
	Artifacts          []ArtifactApplyConfiguration       `json:"artifacts,omitempty"`
	Error              *string                            `json:"error,omitempty"`
	Failure            *FailureApplyConfiguration         `json:"failure,omitempty"`
	StartedAt          *metav1.Time                       `json:"startedAt,omitempty"`
	Conditions         []BuildConditionApplyConfiguration `json:"conditions,omitempty"`
	Duration           *string                            `json:"duration,omitempty"`
}

// BuildStatusApplyConfiguration constructs an declarative configuration of the BuildStatus type for use with
// apply.
func BuildStatus() *BuildStatusApplyConfiguration {
	return &BuildStatusApplyConfiguration{}
}

// WithObservedGeneration sets the ObservedGeneration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ObservedGeneration field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithObservedGeneration(value int64) *BuildStatusApplyConfiguration {
	b.ObservedGeneration = &value
	return b
}

// WithPhase sets the Phase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Phase field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithPhase(value v1.BuildPhase) *BuildStatusApplyConfiguration {
	b.Phase = &value
	return b
}

// WithImage sets the Image field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Image field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithImage(value string) *BuildStatusApplyConfiguration {
	b.Image = &value
	return b
}

// WithDigest sets the Digest field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Digest field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithDigest(value string) *BuildStatusApplyConfiguration {
	b.Digest = &value
	return b
}

// WithBaseImage sets the BaseImage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the BaseImage field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithBaseImage(value string) *BuildStatusApplyConfiguration {
	b.BaseImage = &value
	return b
}

// WithArtifacts adds the given value to the Artifacts field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Artifacts field.
func (b *BuildStatusApplyConfiguration) WithArtifacts(values ...*ArtifactApplyConfiguration) *BuildStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithArtifacts")
		}
		b.Artifacts = append(b.Artifacts, *values[i])
	}
	return b
}

// WithError sets the Error field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Error field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithError(value string) *BuildStatusApplyConfiguration {
	b.Error = &value
	return b
}

// WithFailure sets the Failure field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Failure field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithFailure(value *FailureApplyConfiguration) *BuildStatusApplyConfiguration {
	b.Failure = value
	return b
}

// WithStartedAt sets the StartedAt field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StartedAt field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithStartedAt(value metav1.Time) *BuildStatusApplyConfiguration {
	b.StartedAt = &value
	return b
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *BuildStatusApplyConfiguration) WithConditions(values ...*BuildConditionApplyConfiguration) *BuildStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}

// WithDuration sets the Duration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Duration field is set to the value of the last call.
func (b *BuildStatusApplyConfiguration) WithDuration(value string) *BuildStatusApplyConfiguration {
	b.Duration = &value
	return b
}
