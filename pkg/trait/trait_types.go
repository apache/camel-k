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
	"strings"

	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/property"
)

const (
	True  = "true"
	False = "false"
)

var (
	basePath                  = "/etc/camel"
	confDPath                 = path.Join(basePath, "conf.d")
	sourcesMountPath          = path.Join(basePath, "sources")
	resourcesDefaultMountPath = path.Join(basePath, "resources")
	configResourcesMountPath  = path.Join(confDPath, "_resources")
	configConfigmapsMountPath = path.Join(confDPath, "_configmaps")
	configSecretsMountPath    = path.Join(confDPath, "_secrets")
	serviceBindingsMountPath  = path.Join(confDPath, "_servicebindings")
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

type Comparable interface {
	Matches(Trait) bool
}

type ComparableTrait interface {
	Trait
	Comparable
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

func NewBaseTrait(id string, order int) BaseTrait {
	return BaseTrait{
		TraitID:        ID(id),
		ExecutionOrder: order,
		L:              log.Log.WithName("traits").WithValues("trait", id),
	}
}

// BaseTrait is the root trait with noop implementations for hooks
type BaseTrait struct {
	TraitID ID `json:"-"`
	// Can be used to enable or disable a trait. All traits share this common property.
	Enabled        *bool         `property:"enabled" json:"enabled,omitempty"`
	Client         client.Client `json:"-"`
	ExecutionOrder int           `json:"-"`
	L              log.Logger    `json:"-"`
}

// ID returns the identifier of the trait
func (trait *BaseTrait) ID() ID {
	return trait.TraitID
}

// InjectClient implements client.ClientInject and allows to inject a client into the trait
func (trait *BaseTrait) InjectClient(c client.Client) {
	trait.Client = c
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

// ControllerStrategySelector is the interface for traits that can determine the kind of controller that will run the integration.
type ControllerStrategySelector interface {
	// SelectControllerStrategy tells if the trait with current configuration can select a specific controller to use
	SelectControllerStrategy(*Environment) (*ControllerStrategy, error)
	// ControllerStrategySelectorOrder returns the order (priority) of the controller strategy selector
	ControllerStrategySelectorOrder() int
}

// An Environment provides the context for the execution of the traits
type Environment struct {
	CamelCatalog   *camel.RuntimeCatalog
	RuntimeVersion string
	Catalog        *Catalog
	// The Go standard context for the traits execution
	Ctx context.Context
	// The client to the API server
	Client client.Client
	// The active Platform
	Platform *v1.IntegrationPlatform
	// The current Integration
	Integration *v1.Integration
	// The IntegrationKit associated to the Integration
	IntegrationKit *v1.IntegrationKit
	// The IntegrationKits to be created for the Integration
	IntegrationKits []v1.IntegrationKit
	// The resources owned by the Integration that are applied to the API server
	Resources             *kubernetes.Collection
	PostActions           []func(*Environment) error
	PostStepProcessors    []func(*Environment) error
	PostProcessors        []func(*Environment) error
	BuildTasks            []v1.Task
	ConfiguredTraits      []Trait
	ExecutedTraits        []Trait
	EnvVars               []corev1.EnvVar
	ApplicationProperties map[string]string
	Interceptors          []string
	ServiceBindingSecret  string
}

// ControllerStrategy is used to determine the kind of controller that needs to be created for the integration
type ControllerStrategy string

// List of controller strategies
const (
	ControllerStrategyDeployment     ControllerStrategy = "deployment"
	ControllerStrategyKnativeService ControllerStrategy = "knative-service"
	ControllerStrategyCronJob        ControllerStrategy = "cron-job"

	DefaultControllerStrategy = ControllerStrategyDeployment
)

func (e *Environment) GetTrait(id ID) Trait {
	for _, t := range e.ExecutedTraits {
		if t.ID() == id {
			return t
		}
	}

	return nil
}

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

func (e *Environment) IntegrationInRunningPhases() bool {
	return e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning, v1.IntegrationPhaseError)
}

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
func (e *Environment) DetermineControllerStrategy() (ControllerStrategy, error) {
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

// GetIntegrationPodSpec return the Integration Template Pod Specification, regardless of the deployment strategy
func (e *Environment) GetIntegrationPodSpec() *corev1.PodSpec {
	// Deployment
	deployment := e.Resources.GetDeployment(func(d *appsv1.Deployment) bool {
		return d.Name == e.Integration.Name
	})
	if deployment != nil {
		return &deployment.Spec.Template.Spec
	}

	// Knative service
	knativeService := e.Resources.GetKnativeService(func(s *serving.Service) bool {
		return s.Name == e.Integration.Name
	})
	if knativeService != nil {
		return &knativeService.Spec.Template.Spec.PodSpec
	}

	// Cronjob
	cronJob := e.Resources.GetCronJob(func(c *v1beta1.CronJob) bool {
		return c.Name == e.Integration.Name
	})
	if cronJob != nil {
		return &cronJob.Spec.JobTemplate.Spec.Template.Spec
	}

	return nil
}

func (e *Environment) DetermineCatalogNamespace() string {
	// Catalog is expected to be together with the platform
	if e.Platform != nil && e.Platform.Namespace != "" {
		return e.Platform.Namespace
	}
	if e.Integration != nil && e.Integration.Status.IntegrationKit != nil && e.Integration.Status.IntegrationKit.Namespace != "" {
		return e.Integration.Status.IntegrationKit.Namespace
	}
	if e.Integration != nil && e.Integration.Spec.IntegrationKit != nil && e.Integration.Spec.IntegrationKit.Namespace != "" {
		return e.Integration.Spec.IntegrationKit.Namespace
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Namespace != "" {
		return e.IntegrationKit.Namespace
	}
	if e.Integration != nil && e.Integration.Namespace != "" {
		return e.Integration.Namespace
	}
	return ""
}

func (e *Environment) computeApplicationProperties() (*corev1.ConfigMap, error) {
	// application properties
	applicationProperties, err := property.EncodePropertyFile(e.ApplicationProperties)
	if err != nil {
		return nil, errors.Wrapf(err, "could not compute application properties")
	}

	if applicationProperties != "" {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      e.Integration.Name + "-application-properties",
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					v1.IntegrationLabel:                e.Integration.Name,
					"camel.apache.org/properties.type": "application",
				},
			},
			Data: map[string]string{
				"application.properties": applicationProperties,
			},
		}, nil
	}

	return nil, nil
}

func (e *Environment) addSourcesProperties() {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}
	for i, s := range e.Integration.Sources() {
		srcName := strings.TrimPrefix(s.Name, "/")
		src := "file:" + path.Join(sourcesMountPath, srcName)
		e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].location", i)] = src

		simpleName := srcName
		if strings.Contains(srcName, ".") {
			simpleName = srcName[0:strings.Index(srcName, ".")]
		}
		e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].name", i)] = simpleName

		for pid, p := range s.PropertyNames {
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].property-names[%d]", i, pid)] = p
		}

		if s.Type != "" {
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].type", i)] = string(s.Type)
		}
		if s.InferLanguage() != "" {
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].language", i)] = string(s.InferLanguage())
		}
		if s.Loader != "" {
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].loader", i)] = s.Loader
		}
		if s.Compression {
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].compressed", i)] = "true"
		}

		interceptors := make([]string, 0, len(s.Interceptors))
		if s.Interceptors != nil {
			interceptors = append(interceptors, s.Interceptors...)
		}
		if e.Interceptors != nil {
			interceptors = append(interceptors, e.Interceptors...)
		}
		for intID, interceptor := range interceptors {
			e.ApplicationProperties[fmt.Sprintf("camel.k.sources[%d].interceptors[%d]", i, intID)] = interceptor
		}
	}
}

func (e *Environment) configureVolumesAndMounts(vols *[]corev1.Volume, mnts *[]corev1.VolumeMount) {
	//
	// Volumes :: Sources
	//
	for i, s := range e.Integration.Sources() {
		cmName := fmt.Sprintf("%s-source-%03d", e.Integration.Name, i)
		if s.ContentRef != "" {
			cmName = s.ContentRef
		}
		cmKey := "content"
		if s.ContentKey != "" {
			cmKey = s.ContentKey
		}
		resName := strings.TrimPrefix(s.Name, "/")
		refName := fmt.Sprintf("i-source-%03d", i)
		resPath := path.Join(sourcesMountPath, resName)

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
			ReadOnly:  true,
			SubPath:   resName,
		})
	}

	for i, r := range e.Integration.Resources() {
		if r.Type == v1.ResourceTypeOpenAPI {
			continue
		}

		cmName := fmt.Sprintf("%s-resource-%03d", e.Integration.Name, i)
		refName := fmt.Sprintf("i-resource-%03d", i)
		resName := strings.TrimPrefix(r.Name, "/")
		cmKey := "content"
		resPath := getResourcePath(resName, r.Path, r.Type)

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
			ReadOnly:  true,
			SubPath:   resName,
		})
	}

	if e.Resources != nil {
		e.Resources.VisitConfigMap(func(configMap *corev1.ConfigMap) {
			propertiesType := configMap.Labels["camel.apache.org/properties.type"]
			resName := propertiesType + ".properties"

			var mountPath string
			switch propertiesType {
			case "application":
				mountPath = path.Join(basePath, resName)
			case "user":
				mountPath = path.Join(confDPath, resName)
			}

			if propertiesType != "" {
				*vols = append(*vols, corev1.Volume{
					Name: propertiesType + "-properties",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMap.Name,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  "application.properties",
									Path: resName,
								},
							},
						},
					},
				})

				*mnts = append(*mnts, corev1.VolumeMount{
					Name:      propertiesType + "-properties",
					MountPath: mountPath,
					ReadOnly:  true,
					SubPath:   resName,
				})
			}
		})
	}

	//
	// Volumes :: Additional ConfigMaps
	//
	for _, configmaps := range e.collectConfigurations("configmap") {
		refName := kubernetes.SanitizeLabel(configmaps["value"])

		configmapVolume := corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configmaps["value"],
					},
				},
			},
		}

		// Filter the items selected, if specified
		if configmaps["resourceKey"] != "" {
			configmapVolume.VolumeSource.ConfigMap.Items = []corev1.KeyToPath{
				{
					Key:  configmaps["resourceKey"],
					Path: configmaps["resourceKey"],
				},
			}
		}

		*vols = append(*vols, configmapVolume)

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: getConfigmapMountPoint(configmaps["value"], configmaps["resourceMountPoint"], configmaps["resourceType"]),
			ReadOnly:  true,
		})
	}

	//
	// Volumes :: Additional Secrets
	//
	// append Service Binding secrets
	if len(e.ServiceBindingSecret) > 0 {
		secret := e.ServiceBindingSecret
		refName := kubernetes.SanitizeLabel(secret)

		*vols = append(*vols, corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret,
				},
			},
		})

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: path.Join(serviceBindingsMountPath, strings.ToLower(secret)),
		})
	}

	for _, secret := range e.collectConfigurations("secret") {
		refName := kubernetes.SanitizeLabel(secret["value"])

		secretVolume := corev1.Volume{
			Name: refName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret["value"],
				},
			},
		}

		// Filter the items selected, if specified
		if secret["resourceKey"] != "" {
			secretVolume.VolumeSource.Secret.Items = []corev1.KeyToPath{
				{
					Key:  secret["resourceKey"],
					Path: secret["resourceKey"],
				},
			}
		}

		*vols = append(*vols, secretVolume)

		*mnts = append(*mnts, corev1.VolumeMount{
			Name:      refName,
			MountPath: getSecretMountPoint(secret["value"], secret["resourceMountPoint"], secret["resourceType"]),
			ReadOnly:  true,
		})
	}

	//
	// Volumes :: Additional user provided volumes
	//
	for _, volumeConfig := range e.collectConfigurationValues("volume") {
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

func getResourcePath(resourceName string, maybePath string, resourceType v1.ResourceType) string {
	// If the path is specified, we'll return it
	if maybePath != "" {
		return maybePath
	}
	// otherwise return a default path, according to the resource type
	if resourceType == v1.ResourceTypeData {
		return path.Join(resourcesDefaultMountPath, resourceName)
	}

	// Default, config type
	return path.Join(configResourcesMountPath, resourceName)
}

func getConfigmapMountPoint(resourceName string, maybeMountPoint string, resourceType string) string {
	// If the mount point is specified, we'll return it
	if maybeMountPoint != "" {
		return maybeMountPoint
	}
	if resourceType == "data" {
		return path.Join(resourcesDefaultMountPath, resourceName)
	}

	// Default, config type
	return path.Join(configConfigmapsMountPath, resourceName)
}

func getSecretMountPoint(resourceName string, maybeMountPoint string, resourceType string) string {
	// If the mount point is specified, we'll return it
	if maybeMountPoint != "" {
		return maybeMountPoint
	}
	if resourceType == "data" {
		return path.Join(resourcesDefaultMountPath, resourceName)
	}

	// Default, config type
	return path.Join(configSecretsMountPath, resourceName)
}

func (e *Environment) collectConfigurationValues(configurationType string) []string {
	return collectConfigurationValues(configurationType, e.Platform, e.IntegrationKit, e.Integration)
}

type variable struct {
	Name, Value string
}

func (e *Environment) collectConfigurationPairs(configurationType string) []variable {
	return collectConfigurationPairs(configurationType, e.Platform, e.IntegrationKit, e.Integration)
}

func (e *Environment) collectConfigurations(configurationType string) []map[string]string {
	return collectConfigurations(configurationType, e.Platform, e.IntegrationKit, e.Integration)
}

func (e *Environment) GetIntegrationContainerName() string {
	containerName := defaultContainerName
	dt := e.Catalog.GetTrait(containerTraitID)
	if dt != nil {
		containerName = dt.(*containerTrait).Name
	}
	return containerName
}

func (e *Environment) GetIntegrationContainer() *corev1.Container {
	containerName := e.GetIntegrationContainerName()
	return e.Resources.GetContainerByName(containerName)
}

func (e *Environment) getIntegrationContainerPort() *corev1.ContainerPort {
	container := e.GetIntegrationContainer()
	if container == nil {
		return nil
	}

	portName := ""
	t := e.Catalog.GetTrait(containerTraitID)
	if t != nil {
		portName = t.(*containerTrait).PortName
	}
	if portName == "" {
		portName = defaultContainerPortName
	}

	for i, port := range container.Ports {
		if port.Name == portName {
			return &container.Ports[i]
		}
	}

	return nil
}

// nolint: unused
func (e *Environment) getAllInterceptors() []string {
	res := make([]string, 0)
	util.StringSliceUniqueConcat(&res, e.Interceptors)

	if e.Integration != nil {
		for _, s := range e.Integration.Sources() {
			util.StringSliceUniqueConcat(&res, s.Interceptors)
		}
	}
	return res
}
