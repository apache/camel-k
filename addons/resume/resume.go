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

package resume

import (
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// The Resume trait can be used to manage and configure resume strategies.
//
// This feature is meant to allow quick resume of processing by Camel K instances after they have been restarted.
//
// WARNING: this is an experimental implementation based on the support available on link:/components/next/eips/resume-strategies.html[Camel Core resume strategies].
//
// The Resume trait is disabled by default.
//
// WARNING: The trait is **deprecated** and will removed in future release versions: configure directly the Camel properties as required by the component instead.
//
// The main different from the implementation on Core is that it's not necessary to bind the strategies to the
// registry. This step will be done automatically by Camel K, after resolving the options passed to the trait.
//
// A sample execution of this trait, using the Kafka backend (the only one supported at the moment), would require
// the following trait options:
// -t resume.enabled=true -t resume.resume-path=camel-file-sets -t resume.resume-server="address-of-your-kafka:9092"
//
// +camel-k:trait=resume.
// +camel-k:deprecated=2.5.0.
type Trait struct {
	traitv1.Trait `property:",squash"`
	// Enables automatic configuration of the trait.
	// Deprecated: not in use.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The type of the resume strategy to use
	ResumeStrategy string `property:"resume-strategy,omitempty"`
	// The path used by the resume strategy (this is specific to the resume strategy type)
	ResumePath string `property:"resume-path,omitempty"`
	// The address of the resume server to use (protocol / implementation specific)
	ResumeServer string `property:"resume-server,omitempty"`
	// The adapter-specific policy to use when filling the cache (use: minimizing / maximizing). Check
	// the component documentation if unsure
	CacheFillPolicy string `property:"cache-fill-policy,omitempty"`
}

type resumeTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

const (
	kafkaSingle  = "org.apache.camel.processor.resume.kafka.SingleNodeKafkaResumeStrategy"
	strategyPath = "camel-k-offsets"
)

func NewResumeTrait() trait.Trait {
	return &resumeTrait{
		BaseTrait: trait.NewBaseTrait("resume", trait.TraitOrderBeforeControllerCreation),
	}
}

func (r *resumeTrait) Configure(environment *trait.Environment) (bool, *trait.TraitCondition, error) {
	if !ptr.Deref(r.Enabled, false) {
		return false, nil, nil
	}
	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	condition := trait.NewIntegrationCondition(
		"Resume",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		trait.TraitConfigurationReason,
		"Resume trait is deprecated and may be removed in future version: "+
			"configure directly the Camel properties as required by the component instead",
	)

	return ptr.Deref(r.Enabled, false), condition, nil
}

func (r *resumeTrait) Apply(environment *trait.Environment) error {
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityResumeKafka)
	}

	if environment.IntegrationInRunningPhases() {
		environment.ApplicationProperties["customizer.resume.enabled"] = "true"
		environment.ApplicationProperties["customizer.resume.resumeStrategy"] = r.getResumeStrategy()
		environment.ApplicationProperties["customizer.resume.resumePath"] = r.getResumePath()
		environment.ApplicationProperties["customizer.resume.resumeServer"] = r.ResumeServer
		environment.ApplicationProperties["customizer.resume.cacheFillPolicy"] = r.CacheFillPolicy
	}

	return nil
}

func (r *resumeTrait) getResumeStrategy() string {
	if r.ResumeStrategy == "" {
		return kafkaSingle
	}

	return r.ResumeStrategy
}

func (r *resumeTrait) getResumePath() string {
	if r.ResumePath == "" {
		return strategyPath
	}

	return r.ResumePath
}
