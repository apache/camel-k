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

package v1alpha1

import (
	"github.com/apache/camel-k/pkg/util"
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strings"
)

// ConfigurationSpec --
type ConfigurationSpec struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationList --
type IntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Integration `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Integration --
type Integration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              IntegrationSpec   `json:"spec"`
	Status            IntegrationStatus `json:"status,omitempty"`
}

// IntegrationSpec --
type IntegrationSpec struct {
	Replicas      *int32                          `json:"replicas,omitempty"`
	Sources       []SourceSpec                    `json:"sources,omitempty"`
	Resources     []ResourceSpec                  `json:"resources,omitempty"`
	Context       string                          `json:"context,omitempty"`
	Dependencies  []string                        `json:"dependencies,omitempty"`
	Profile       TraitProfile                    `json:"profile,omitempty"`
	Traits        map[string]IntegrationTraitSpec `json:"traits,omitempty"`
	Configuration []ConfigurationSpec             `json:"configuration,omitempty"`
	Repositories  []string                        `json:"repositories,omitempty"`
}

// AddSource --
func (is *IntegrationSpec) AddSource(name string, content string, language Language) {
	is.Sources = append(is.Sources, NewSourceSpec(name, content, language))
}

// AddSources --
func (is *IntegrationSpec) AddSources(sources ...SourceSpec) {
	is.Sources = append(is.Sources, sources...)
}

// AddResources --
func (is *IntegrationSpec) AddResources(resources ...ResourceSpec) {
	is.Resources = append(is.Resources, resources...)
}

// AddConfiguration --
func (is *IntegrationSpec) AddConfiguration(confType string, confValue string) {
	is.Configuration = append(is.Configuration, ConfigurationSpec{
		Type:  confType,
		Value: confValue,
	})
}

// AddDependency --
func (is *IntegrationSpec) AddDependency(dependency string) {
	switch {
	case strings.HasPrefix(dependency, "mvn:"):
		util.StringSliceUniqueAdd(&is.Dependencies, dependency)
	case strings.HasPrefix(dependency, "file:"):
		util.StringSliceUniqueAdd(&is.Dependencies, dependency)
	case strings.HasPrefix(dependency, "camel-"):
		util.StringSliceUniqueAdd(&is.Dependencies, "camel:"+strings.TrimPrefix(dependency, "camel-"))
	}
}

// DataSpec --
type DataSpec struct {
	Name        string `json:"name,omitempty"`
	Content     string `json:"content,omitempty"`
	Compression bool   `json:"compression,omitempty"`
}

// ResourceSpec --
type ResourceSpec struct {
	DataSpec
}

// SourceSpec --
type SourceSpec struct {
	DataSpec
	Language Language `json:"language,omitempty"`
}

// Language --
type Language string

const (
	// LanguageJavaSource --
	LanguageJavaSource Language = "java"
	// LanguageJavaClass --
	LanguageJavaClass Language = "class"
	// LanguageGroovy --
	LanguageGroovy Language = "groovy"
	// LanguageJavaScript --
	LanguageJavaScript Language = "js"
	// LanguageXML --
	LanguageXML Language = "xml"
	// LanguageKotlin --
	LanguageKotlin Language = "kts"
	// LanguageYamlFlow --
	LanguageYamlFlow Language = "flow"
)

// Languages is the list of all supported languages
var Languages = []Language{
	LanguageJavaSource,
	LanguageJavaClass,
	LanguageJavaScript,
	LanguageGroovy,
	LanguageJavaScript,
	LanguageKotlin,
	LanguageYamlFlow,
}

// A IntegrationTraitSpec contains the configuration of a trait
type IntegrationTraitSpec struct {
	Configuration map[string]string `json:"configuration,omitempty"`
}

// Decode the trait configuration to a type safe struct
func (in *IntegrationTraitSpec) Decode(target interface{}) error {
	md := mapstructure.Metadata{}

	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Metadata:         &md,
			WeaklyTypedInput: true,
			TagName:          "property",
			Result:           &target,
		},
	)

	if err != nil {
		return err
	}

	return decoder.Decode(in.Configuration)
}

// IntegrationStatus --
type IntegrationStatus struct {
	Phase        IntegrationPhase `json:"phase,omitempty"`
	Digest       string           `json:"digest,omitempty"`
	Image        string           `json:"image,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty"`
	Context      string           `json:"context,omitempty"`
}

// IntegrationPhase --
type IntegrationPhase string

const (
	// IntegrationKind --
	IntegrationKind string = "Integration"

	// IntegrationPhaseBuildingContext --
	IntegrationPhaseBuildingContext IntegrationPhase = "Building Context"
	// IntegrationPhaseBuildingImage --
	IntegrationPhaseBuildingImage IntegrationPhase = "Building Image"
	// IntegrationPhaseDeploying --
	IntegrationPhaseDeploying IntegrationPhase = "Deploying"
	// IntegrationPhaseRunning --
	IntegrationPhaseRunning IntegrationPhase = "Running"
	// IntegrationPhaseError --
	IntegrationPhaseError IntegrationPhase = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationContextList --
type IntegrationContextList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IntegrationContext `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationContext --
type IntegrationContext struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              IntegrationContextSpec   `json:"spec"`
	Status            IntegrationContextStatus `json:"status,omitempty"`
}

// IntegrationContextSpec --
type IntegrationContextSpec struct {
	Dependencies  []string                        `json:"dependencies,omitempty"`
	Profile       TraitProfile                    `json:"profile,omitempty"`
	Traits        map[string]IntegrationTraitSpec `json:"traits,omitempty"`
	Configuration []ConfigurationSpec             `json:"configuration,omitempty"`
	Repositories  []string                        `json:"repositories,omitempty"`
}

// IntegrationContextStatus --
type IntegrationContextStatus struct {
	Phase       IntegrationContextPhase `json:"phase,omitempty"`
	Image       string                  `json:"image,omitempty"`
	PublicImage string                  `json:"publicImage,omitempty"`
	Digest      string                  `json:"digest,omitempty"`
	Artifacts   []Artifact              `json:"artifacts,omitempty"`
}

// ImageForIntegration returns the image to use when using it for running an integration
func (c IntegrationContext) ImageForIntegration() string {
	if c.Status.PublicImage != "" {
		return c.Status.PublicImage
	}
	return c.Status.Image
}

// IntegrationContextPhase --
type IntegrationContextPhase string

const (
	// IntegrationContextKind --
	IntegrationContextKind string = "IntegrationContext"

	// IntegrationContextTypePlatform --
	IntegrationContextTypePlatform = "platform"

	// IntegrationContextPhaseBuilding --
	IntegrationContextPhaseBuilding IntegrationContextPhase = "Building"
	// IntegrationContextPhaseReady --
	IntegrationContextPhaseReady IntegrationContextPhase = "Ready"
	// IntegrationContextPhaseError --
	IntegrationContextPhaseError IntegrationContextPhase = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationPlatformList --
type IntegrationPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IntegrationPlatform `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IntegrationPlatform --
type IntegrationPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              IntegrationPlatformSpec   `json:"spec"`
	Status            IntegrationPlatformStatus `json:"status,omitempty"`
}

// IntegrationPlatformSpec --
type IntegrationPlatformSpec struct {
	Cluster IntegrationPlatformCluster   `json:"cluster,omitempty"`
	Profile TraitProfile                 `json:"profile,omitempty"`
	Build   IntegrationPlatformBuildSpec `json:"build,omitempty"`
}

// IntegrationPlatformCluster is the kind of orchestration cluster the platform is installed into
type IntegrationPlatformCluster string

const (
	// IntegrationPlatformClusterOpenShift is used when targeting a OpenShift cluster
	IntegrationPlatformClusterOpenShift = "OpenShift"
	// IntegrationPlatformClusterKubernetes is used when targeting a Kubernetes cluster
	IntegrationPlatformClusterKubernetes = "Kubernetes"
)

// TraitProfile represents lists of traits that are enabled for the specific installation/integration
type TraitProfile string

const (
	// TraitProfileOpenShift is used by default on OpenShift clusters
	TraitProfileOpenShift = "OpenShift"
	// TraitProfileKubernetes is used by default on Kubernetes clusters
	TraitProfileKubernetes = "Kubernetes"
	// TraitProfileKnative is used by default on OpenShift/Kubernetes clusters powered by Knative
	TraitProfileKnative = "Knative"
)

var allTraitProfiles = []TraitProfile{TraitProfileOpenShift, TraitProfileKubernetes, TraitProfileKnative}

// IntegrationPlatformBuildSpec contains platform related build information
type IntegrationPlatformBuildSpec struct {
	PublishStrategy IntegrationPlatformBuildPublishStrategy `json:"publishStrategy,omitempty"`
	Registry        string                                  `json:"registry,omitempty"`
	Organization    string                                  `json:"organization,omitempty"`
	PushSecret      string                                  `json:"pushSecret,omitempty"`
	CamelVersion    string                                  `json:"camelVersion,omitempty"`
	Properties      map[string]string                       `json:"properties,omitempty"`
	Repositories    []string                                `json:"repositories,omitempty"`
}

// IntegrationPlatformBuildPublishStrategy enumerates all implemented build strategies
type IntegrationPlatformBuildPublishStrategy string

const (
	// IntegrationPlatformBuildPublishStrategyS2I performs a OpenShift binary S2I build
	IntegrationPlatformBuildPublishStrategyS2I = "S2I"

	// IntegrationPlatformBuildPublishStrategyKaniko performs
	IntegrationPlatformBuildPublishStrategyKaniko = "Kaniko"
)

// IntegrationPlatformStatus --
type IntegrationPlatformStatus struct {
	Phase IntegrationPlatformPhase `json:"phase,omitempty"`
}

// IntegrationPlatformPhase --
type IntegrationPlatformPhase string

const (
	// IntegrationPlatformKind --
	IntegrationPlatformKind string = "IntegrationPlatform"

	// IntegrationPlatformPhaseCreating --
	IntegrationPlatformPhaseCreating IntegrationPlatformPhase = "Creating"
	// IntegrationPlatformPhaseStarting --
	IntegrationPlatformPhaseStarting IntegrationPlatformPhase = "Starting"
	// IntegrationPlatformPhaseReady --
	IntegrationPlatformPhaseReady IntegrationPlatformPhase = "Ready"
	// IntegrationPlatformPhaseError --
	IntegrationPlatformPhaseError IntegrationPlatformPhase = "Error"
	// IntegrationPlatformPhaseDuplicate --
	IntegrationPlatformPhaseDuplicate IntegrationPlatformPhase = "Duplicate"
)

// Artifact --
type Artifact struct {
	ID       string `json:"id" yaml:"id"`
	Location string `json:"location,omitempty" yaml:"location,omitempty"`
	Target   string `json:"target,omitempty" yaml:"target,omitempty"`
}

func (in *Artifact) String() string {
	return in.ID
}

// Flow --
type Flow struct {
	Steps []Step `json:"steps"`
}

// Flows are collections of Flow
type Flows []Flow

// Step --
type Step struct {
	Kind string `json:"kind"`
	URI  string `json:"uri"`
}
