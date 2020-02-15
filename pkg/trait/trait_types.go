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

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controller "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
)

// True --
const True = "true"

var (
	// BasePath --
	BasePath = "/etc/camel"

	// ConfPath --
	ConfPath = path.Join(BasePath, "conf")

	// ConfdPath --
	ConfdPath = path.Join(BasePath, "conf.d")

	// SourcesMountPath --
	SourcesMountPath = path.Join(BasePath, "sources")

	// ResourcesMountPath --
	ResourcesMountPath = path.Join(BasePath, "resources")

	// ConfigMapsMountPath --
	ConfigMapsMountPath = path.Join(ConfdPath, "_configmaps")

	// SecretsMountPath --
	SecretsMountPath = path.Join(ConfdPath, "_secrets")
)

// Identifiable represent an identifiable type
type Identifiable interface {
	ID() ID
}

// ID uniquely identifies a trait
type ID string

// Trait is the interface of all traits
type Trait interface {
	Identifiable
	client.Injectable

	// InjectContext to inject a context
	InjectContext(context.Context)

	// Configure the trait
	Configure(environment *Environment) (bool, error)

	// Apply executes a customization of the Environment
	Apply(environment *Environment) error

	// InfluencesKit determines if the trait has any influence on Integration Kits
	InfluencesKit() bool

	// IsPlatformTrait marks all fundamental traits that allow the platform to work
	IsPlatformTrait() bool

	// RequiresIntegrationPlatform indicates that the trait cannot work without an integration platform set
	RequiresIntegrationPlatform() bool

	// IsAllowedInProfile tels if the trait supports the given profile
	IsAllowedInProfile(v1.TraitProfile) bool

	// Order is the order in which the trait should be executed in the normal flow
	Order() int
}

// A list of named orders, useful for correctly binding addons
const (
	// TraitOrderBeforeControllerCreation can be used to inject configuration such as properties and environment variables
	// into the running integration, before the actual controller is created.
	TraitOrderBeforeControllerCreation = 850
	// TraitOrderControllerSelection can be used if you intend to provide an alternative controller for the integration
	// (e.g. Deployment, CronJob, ...).
	TraitOrderControllerSelection = 950
	// TraitOrderPostProcessResources is executed after all other traits are executed (except for important core traits such as
	// the "owner" trait), so it can be used to post-process generated resources before their actual creation.
	TraitOrderPostProcessResources = 2450
)

/* Base trait */

// NewBaseTrait --
func NewBaseTrait(id string, order int) BaseTrait {
	return BaseTrait{
		TraitID:        ID(id),
		ExecutionOrder: order,
		L:              log.Log.WithName("traits").WithValues("trait", id),
	}
}

// BaseTrait is the root trait with noop implementations for hooks
type BaseTrait struct {
	TraitID ID
	// Can be used to enable or disable a trait. All traits share this common property.
	Enabled        *bool `property:"enabled"`
	Client         client.Client
	Ctx            context.Context
	ExecutionOrder int
	L              log.Logger
}

// ID returns the identifier of the trait
func (trait *BaseTrait) ID() ID {
	return trait.TraitID
}

// InjectClient implements client.ClientInject and allows to inject a client into the trait
func (trait *BaseTrait) InjectClient(c client.Client) {
	trait.Client = c
}

// InjectContext allows to inject a context into the trait
func (trait *BaseTrait) InjectContext(ctx context.Context) {
	trait.Ctx = ctx
}

// InfluencesKit determines if the trait has any influence on Integration Kits
func (trait *BaseTrait) InfluencesKit() bool {
	return false
}

// IsPlatformTrait marks all fundamental traits that allow the platform to work.
func (trait *BaseTrait) IsPlatformTrait() bool {
	return false
}

// RequiresIntegrationPlatform indicates that the trait cannot work without an integration platform set
func (trait *BaseTrait) RequiresIntegrationPlatform() bool {
	// All traits require a platform by default
	return true
}

// IsAllowedInProfile returns true for any profile by default
func (trait *BaseTrait) IsAllowedInProfile(v1.TraitProfile) bool {
	return true
}

// Order contains the order value provided during initialization
func (trait *BaseTrait) Order() int {
	return trait.ExecutionOrder
}

/* ControllerStrategySelector */

// ControllerStrategySelector is the interface for traits that can determine the kind of controller that will run the integration.
type ControllerStrategySelector interface {
	// SelectControllerStrategy tells if the trait with current configuration can select a specific controller to use
	SelectControllerStrategy(*Environment) (*ControllerStrategy, error)
	// ControllerStrategySelectorOrder returns the order (priority) of the controller strategy selector
	ControllerStrategySelectorOrder() int
}

/* Environment */

// A Environment provides the context where the trait is executed
type Environment struct {
	CamelCatalog     *camel.RuntimeCatalog
	RuntimeVersion   string
	Catalog          *Catalog
	C                context.Context
	Client           client.Client
	Platform         *v1.IntegrationPlatform
	IntegrationKit   *v1.IntegrationKit
	Integration      *v1.Integration
	Resources        *kubernetes.Collection
	PostActions      []func(*Environment) error
	PostProcessors   []func(*Environment) error
	BuildTasks       []v1.Task
	ConfiguredTraits []Trait
	ExecutedTraits   []Trait
	EnvVars          []corev1.EnvVar
}

// ControllerStrategy is used to determine the kind of controller that needs to be created for the integration
type ControllerStrategy string

// List of controller strategies
const (
	ControllerStrategyDeployment     ControllerStrategy = "deployment"
	ControllerStrategyKnativeService ControllerStrategy = "knative-service"
	ControllerStrategyCronJob        ControllerStrategy = "cron-job"

	DefaultControllerStrategy ControllerStrategy = ControllerStrategyDeployment
)

// GetTrait --
func (e *Environment) GetTrait(id ID) Trait {
	for _, t := range e.ExecutedTraits {
		if t.ID() == id {
			return t
		}
	}

	return nil
}

// IntegrationInPhase --
func (e *Environment) IntegrationInPhase(phases ...v1.IntegrationPhase) bool {
	if e.Integration == nil {
		return false
	}

	for _, phase := range phases {
		if e.Integration.Status.Phase == phase {
			return true
		}
	}

	return false
}

// IntegrationKitInPhase --
func (e *Environment) IntegrationKitInPhase(phases ...v1.IntegrationKitPhase) bool {
	if e.IntegrationKit == nil {
		return false
	}

	for _, phase := range phases {
		if e.IntegrationKit.Status.Phase == phase {
			return true
		}
	}

	return false
}

// InPhase --
func (e *Environment) InPhase(c v1.IntegrationKitPhase, i v1.IntegrationPhase) bool {
	return e.IntegrationKitInPhase(c) && e.IntegrationInPhase(i)
}

// DetermineProfile determines the TraitProfile of the environment.
// First looking at the Integration.Spec for a Profile,
// next looking at the IntegrationKit.Spec
// and lastly the Platform Profile
func (e *Environment) DetermineProfile() v1.TraitProfile {
	if e.Integration != nil {
		if e.Integration.Status.Profile != "" {
			return e.Integration.Status.Profile
		}
		if e.Integration.Spec.Profile != "" {
			return e.Integration.Spec.Profile
		}
	}

	if e.IntegrationKit != nil && e.IntegrationKit.Spec.Profile != "" {
		return e.IntegrationKit.Spec.Profile
	}

	if e.Platform != nil {
		return platform.GetProfile(e.Platform)
	}

	return v1.DefaultTraitProfile
}

// DetermineControllerStrategy determines the type of controller that should be used for the integration
func (e *Environment) DetermineControllerStrategy(ctx context.Context, c controller.Reader) (ControllerStrategy, error) {
	defaultStrategy := DefaultControllerStrategy
	for _, creator := range e.getControllerStrategyChoosers() {
		if strategy, err := creator.SelectControllerStrategy(e); err != nil {
			return defaultStrategy, err
		} else if strategy != nil {
			return *strategy, nil
		}
	}

	return defaultStrategy, nil
}

func (e *Environment) getControllerStrategyChoosers() (res []ControllerStrategySelector) {
	for _, t := range e.ConfiguredTraits {
		if cc, ok := t.(ControllerStrategySelector); ok {
			res = append(res, cc)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ControllerStrategySelectorOrder() < res[j].ControllerStrategySelectorOrder()
	})
	return res
}

// DetermineNamespace --
func (e *Environment) DetermineNamespace() string {
	if e.Integration != nil && e.Integration.Namespace != "" {
		return e.Integration.Namespace
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Namespace != "" {
		return e.IntegrationKit.Namespace
	}
	if e.Platform != nil && e.Platform.Namespace != "" {
		return e.Platform.Namespace
	}

	return ""
}

// ComputeConfigMaps --
func (e *Environment) ComputeConfigMaps() []runtime.Object {
	sources := e.Integration.Sources()
	maps := make([]runtime.Object, 0, len(sources)+1)

	// combine properties of integration with kit, integration
	// properties have the priority
	properties := ""

	for key, val := range e.CollectConfigurationPairs("property") {
		properties += fmt.Sprintf("%s=%s\n", key, val)
	}

	maps = append(
		maps,
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      e.Integration.Name + "-properties",
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
			},
			Data: map[string]string{
				"application.properties": properties,
			},
		},
	)

	for i, s := range sources {
		if s.ContentRef != "" {
			continue
		}

		cm := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-source-%03d", e.Integration.Name, i),
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
				Annotations: map[string]string{
					"camel.apache.org/source.language":    string(s.InferLanguage()),
					"camel.apache.org/source.loader":      s.Loader,
					"camel.apache.org/source.name":        s.Name,
					"camel.apache.org/source.compression": strconv.FormatBool(s.Compression),
				},
			},
			Data: map[string]string{
				"content": s.Content,
			},
		}

		maps = append(maps, &cm)
	}

	for i, r := range e.Integration.Spec.Resources {
		if r.Type != v1.ResourceTypeData {
			continue
		}
		if r.ContentRef != "" {
			continue
		}

		cmKey := "content"
		if r.ContentKey != "" {
			cmKey = r.ContentKey
		}

		cm := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-resource-%03d", e.Integration.Name, i),
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
				Annotations: map[string]string{
					"camel.apache.org/resource.name":        r.Name,
					"camel.apache.org/resource.compression": strconv.FormatBool(r.Compression),
				},
			},
			Data: map[string]string{
				cmKey: r.Content,
			},
		}

		maps = append(maps, &cm)
	}

	return maps
}

// ComputeSourcesURI --
func (e *Environment) ComputeSourcesURI() []string {
	sources := e.Integration.Sources()
	paths := make([]string, 0, len(sources))

	for i, s := range sources {
		root := path.Join(SourcesMountPath, fmt.Sprintf("i-source-%03d", i))

		srcName := strings.TrimPrefix(s.Name, "/")
		src := path.Join(root, srcName)
		src = "file:" + src

		params := make([]string, 0)
		if s.InferLanguage() != "" {
			params = append(params, "language="+string(s.InferLanguage()))
		}
		if s.Loader != "" {
			params = append(params, "loader="+s.Loader)
		}
		if s.Compression {
			params = append(params, "compression=true")
		}

		if len(params) > 0 {
			src = fmt.Sprintf("%s?%s", src, strings.Join(params, "&"))
		}

		paths = append(paths, src)
	}

	return paths
}

// ConfigureVolumesAndMounts --
func (e *Environment) ConfigureVolumesAndMounts(vols *[]corev1.Volume, mnts *[]corev1.VolumeMount) {
	//
	// Volumes :: Sources
	//

	for i, s := range e.Integration.Sources() {
		cmName := fmt.Sprintf("%s-source-%03d", e.Integration.Name, i)
		refName := fmt.Sprintf("i-source-%03d", i)
		resName := strings.TrimPrefix(s.Name, "/")
		resPath := path.Join(SourcesMountPath, refName)

		if s.ContentRef != "" {
			cmName = s.ContentRef
		}

		*vols = append(*vols, corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "content",
							Path: resName,
						},
					},
				},
			},
		})

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: resPath,
		})
	}

	for i, r := range e.Integration.Resources() {
		if r.Type != v1.ResourceTypeData {
			continue
		}

		cmName := fmt.Sprintf("%s-resource-%03d", e.Integration.Name, i)
		refName := fmt.Sprintf("i-resource-%03d", i)
		resName := strings.TrimPrefix(r.Name, "/")
		cmKey := "content"
		resPath := path.Join(ResourcesMountPath, refName)

		if r.ContentRef != "" {
			cmName = r.ContentRef
		}
		if r.ContentKey != "" {
			cmKey = r.ContentKey
		}
		if r.MountPath != "" {
			resPath = r.MountPath
		}

		*vols = append(*vols, corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  cmKey,
							Path: resName,
						},
					},
				},
			},
		})

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: resPath,
		})
	}

	//
	// Volumes :: Properties
	//

	*vols = append(*vols, corev1.Volume{
		Name: "integration-properties",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: e.Integration.Name + "-properties",
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "application.properties",
						Path: "application.properties",
					},
				},
			},
		},
	})

	*mnts = append(*mnts, corev1.VolumeMount{
		Name:      "integration-properties",
		MountPath: ConfPath,
	})

	//
	// Volumes :: Additional ConfigMaps
	//

	for _, cmName := range e.CollectConfigurationValues("configmap") {
		refName := kubernetes.SanitizeLabel(cmName)

		*vols = append(*vols, corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmName,
					},
				},
			},
		})

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: path.Join(ConfigMapsMountPath, strings.ToLower(cmName)),
		})
	}

	//
	// Volumes :: Additional Secrets
	//

	for _, secretName := range e.CollectConfigurationValues("secret") {
		refName := kubernetes.SanitizeLabel(secretName)

		*vols = append(*vols, corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: path.Join(SecretsMountPath, strings.ToLower(secretName)),
		})
	}

	//
	// Volumes :: Additional user provided volumes
	//
	for _, volumeConfig := range e.CollectConfigurationValues("volume") {
		configParts := strings.Split(volumeConfig, ":")

		if len(configParts) != 2 {
			continue
		}

		pvcName := configParts[0]
		mountPath := configParts[1]
		volumeName := pvcName + "-data"

		*vols = append(*vols, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
		})
	}
}

// CollectConfigurationValues --
func (e *Environment) CollectConfigurationValues(configurationType string) []string {
	return CollectConfigurationValues(configurationType, e.Platform, e.IntegrationKit, e.Integration)
}

// CollectConfigurationPairs --
func (e *Environment) CollectConfigurationPairs(configurationType string) map[string]string {
	return CollectConfigurationPairs(configurationType, e.Platform, e.IntegrationKit, e.Integration)
}

func (e *Environment) getIntegrationContainer() *corev1.Container {
	containerName := defaultContainerName
	dt := e.Catalog.GetTrait(containerTraitID)
	if dt != nil {
		containerName = dt.(*containerTrait).Name
	}

	return e.Resources.GetContainerByName(containerName)
}
