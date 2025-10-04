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
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/metadata"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

const (
	sourceLanguageAnnotation    = "camel.apache.org/source.language"
	sourceLoaderAnnotation      = "camel.apache.org/source.loader"
	sourceNameAnnotation        = "camel.apache.org/source.name"
	sourceCompressionAnnotation = "camel.apache.org/source.compression"

	defaultContainerPortName = "http"
	// Knative does not want name=http, it only supports http1 (HTTP/1) and h2c (HTTP/2)
	// https://github.com/knative/specs/blob/main/specs/serving/runtime-contract.md#protocols-and-ports
	defaultKnativeContainerPortName = "h2c"

	secretStorageType    = "secret"
	configmapStorageType = "configmap"
	pvcStorageType       = "pvc"
	emptyDirStorageType  = "emptyDir"
)

var capabilityDynamicProperty = regexp.MustCompile(`(\$\{([^}]*)\})`)

// Identifiable represent an identifiable type.
type Identifiable interface {
	ID() ID
}

// ID uniquely identifies a trait.
type ID string

// Trait is the interface of all traits.
type Trait interface {
	Identifiable
	client.Injectable

	// Configure the trait
	Configure(environment *Environment) (bool, *TraitCondition, error)

	// Apply executes a customization of the Environment
	Apply(environment *Environment) error

	// InfluencesKit determines if the trait has any influence on Integration Kits
	InfluencesKit() bool

	// IsPlatformTrait marks all fundamental traits that allow the platform to work
	IsPlatformTrait() bool

	// RequiresIntegrationPlatform indicates that the trait cannot work without an integration platform set
	RequiresIntegrationPlatform() bool

	// IsAllowedInProfile tells if the trait supports the given profile
	IsAllowedInProfile(traitProfile v1.TraitProfile) bool

	// Order is the order in which the trait should be executed in the normal flow
	Order() int
}

// Comparable is the interface exposing comparable funcs.
type Comparable interface {
	Matches(trait Trait) bool
}

// ComparableTrait is the interface used to compare two traits between them.
type ComparableTrait interface {
	Trait
	Comparable
}

// A list of named orders, useful for traits sorting.
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

func NewBasePlatformTrait(id string, order int) BasePlatformTrait {
	return BasePlatformTrait{
		BaseTrait{
			TraitID:        ID(id),
			ExecutionOrder: order,
			L:              log.Log.WithName("traits").WithValues("trait", id),
		},
	}
}

// BaseTrait is the root trait with noop implementations for hooks.
type BaseTrait struct {
	TraitID        ID            `json:"-"`
	Client         client.Client `json:"-"`
	ExecutionOrder int           `json:"-"`
	L              log.Logger    `json:"-"`
}

// ID returns the identifier of the trait.
func (trait *BaseTrait) ID() ID {
	return trait.TraitID
}

// InjectClient implements client.ClientInject and allows to inject a client into the trait.
func (trait *BaseTrait) InjectClient(c client.Client) {
	trait.Client = c
}

// InfluencesKit determines if the trait has any influence on Integration Kits.
func (trait *BaseTrait) InfluencesKit() bool {
	return false
}

// IsPlatformTrait marks all fundamental traits that allow the platform to work.
func (trait *BaseTrait) IsPlatformTrait() bool {
	return false
}

// RequiresIntegrationPlatform indicates that the trait cannot work without an integration platform set.
func (trait *BaseTrait) RequiresIntegrationPlatform() bool {
	// All traits require a platform by default
	return true
}

// IsAllowedInProfile returns true for any profile by default.
func (trait *BaseTrait) IsAllowedInProfile(v1.TraitProfile) bool {
	return true
}

// Order contains the order value provided during initialization.
func (trait *BaseTrait) Order() int {
	return trait.ExecutionOrder
}

// BasePlatformTrait is the root for platform traits with noop implementations for hooks.
type BasePlatformTrait struct {
	BaseTrait
}

// IsPlatformTrait marks all fundamental traits that allow the platform to work.
func (trait *BasePlatformTrait) IsPlatformTrait() bool {
	return true
}

// ControllerStrategySelector is the interface for traits that can determine the kind of controller that will run the integration.
type ControllerStrategySelector interface {
	// SelectControllerStrategy tells if the trait with current configuration can select a specific controller to use
	SelectControllerStrategy(env *Environment) (*ControllerStrategy, error)
	// ControllerStrategySelectorOrder returns the order (priority) of the controller strategy selector
	ControllerStrategySelectorOrder() int
}

// An Environment provides the context for the execution of the traits.
//
//nolint:containedctx
type Environment struct {
	CamelCatalog *camel.RuntimeCatalog
	Catalog      *Catalog
	// The Go standard context for the traits execution
	Ctx context.Context
	// The client to the API server
	Client client.Client
	// The active Platform
	Platform *v1.IntegrationPlatform
	// The active IntegrationProfile
	IntegrationProfile *v1.IntegrationProfile
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
	Pipeline              []v1.Task
	ConfiguredTraits      []Trait
	ExecutedTraits        []Trait
	EnvVars               []corev1.EnvVar
	ApplicationProperties map[string]string
}

// ControllerStrategy is used to determine the kind of controller that needs to be created for the integration.
type ControllerStrategy string

// List of controller strategies.
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

func (e *Environment) PlatformInPhase(phases ...v1.IntegrationPlatformPhase) bool {
	if e.Platform == nil {
		return false
	}

	for _, phase := range phases {
		if e.Platform.Status.Phase == phase {
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
// and lastly the Platform Profile.
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
		return platform.GetTraitProfile(e.Platform)
	}

	return v1.DefaultTraitProfile
}

// DetermineControllerStrategy determines the type of controller that should be used for the integration.
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

// determineDefaultContainerPortName determines the default port name, according the controller strategy used.
func (e *Environment) determineDefaultContainerPortName() string {
	controller, err := e.DetermineControllerStrategy()
	if err != nil {
		log.WithValues("Function", "trait.determineDefaultContainerPortName").Errorf(err, "could not determine controller strategy, using default deployment container name")
		return defaultContainerPortName
	}
	if controller == ControllerStrategyKnativeService {
		return defaultKnativeContainerPortName
	}
	return defaultContainerPortName
}

func (e *Environment) getControllerStrategyChoosers() []ControllerStrategySelector {
	var res []ControllerStrategySelector
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

// GetIntegrationPodSpec return the Integration Template Pod Specification, regardless of the deployment strategy.
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
	cronJob := e.Resources.GetCronJob(func(c *batchv1.CronJob) bool {
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

func getVolume(volName, storageType, storageName, filterKey, filterValue string) *corev1.Volume {
	items := convertToKeyToPath(filterKey, filterValue)
	volume := corev1.Volume{
		Name:         volName,
		VolumeSource: corev1.VolumeSource{},
	}
	switch storageType {
	case configmapStorageType:
		volume.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: storageName,
			},
			Items: items,
		}
	case secretStorageType:
		volume.Secret = &corev1.SecretVolumeSource{
			SecretName: storageName,
			Items:      items,
		}
	case pvcStorageType:
		volume.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: storageName,
		}
	}

	return &volume
}

func getMount(volName, mountPath, subPath string, readOnly bool) *corev1.VolumeMount {
	mount := corev1.VolumeMount{
		Name:      volName,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
	if subPath != "" {
		mount.SubPath = subPath
	}

	return &mount
}

func convertToKeyToPath(k, v string) []corev1.KeyToPath {
	if k == "" {
		return nil
	}
	if v == "" {
		v = k
	}
	kp := []corev1.KeyToPath{
		{
			Key:  k,
			Path: v,
		},
	}

	return kp
}

func getMountPoint(resourceName string, mountPoint string, storagetype, resourceType string) string {
	if mountPoint != "" {
		return mountPoint
	}
	if resourceType == "data" {
		defaultResourceMountPoint := camel.ResourcesConfigmapsMountPath
		if storagetype == secretStorageType {
			defaultResourceMountPoint = camel.ResourcesSecretsMountPath
		}
		return filepath.Join(defaultResourceMountPoint, resourceName)
	}
	defaultMountPoint := camel.ConfigConfigmapsMountPath
	if storagetype == secretStorageType {
		defaultMountPoint = camel.ConfigSecretsMountPath
	}

	return filepath.Join(defaultMountPoint, resourceName)
}

type variable struct {
	Name, Value string
}

func (e *Environment) collectConfigurationPairs(configurationType string) []variable {
	return collectConfigurationPairs(configurationType, e.Platform, e.IntegrationKit, e.Integration)
}

func (e *Environment) GetIntegrationContainerName() string {
	containerName := defaultContainerName

	if dt := e.Catalog.GetTrait(containerTraitID); dt != nil {
		if ct, ok := dt.(*containerTrait); ok {
			containerName = ct.getContainerName()
		}
	}
	return containerName
}

// Indicates whether the given source is embedded in the final binary.
func (e *Environment) isEmbedded(source v1.SourceSpec) bool {
	if dt := e.Catalog.GetTrait(quarkusTraitID); dt != nil {
		if qt, ok := dt.(*quarkusTrait); ok {
			return qt.isEmbedded(e, source)
		}
	}
	return false
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

	// User specified port name
	portName := ""
	if t := e.Catalog.GetTrait(containerTraitID); t != nil {
		if ct, ok := t.(*containerTrait); ok {
			portName = ct.PortName
		}
	}

	// default port name (may change according the controller strategy, ie Knative)
	if portName == "" {
		portName = e.determineDefaultContainerPortName()
	}

	for i, port := range container.Ports {
		if port.Name == portName {
			return &container.Ports[i]
		}
	}

	return nil
}

// createContainerPort creates a new container port with values taken from Container trait or default.
func (e *Environment) createContainerPort() *corev1.ContainerPort {
	var name string
	var port int32

	if t := e.Catalog.GetTrait(containerTraitID); t != nil {
		if ct, ok := t.(*containerTrait); ok {
			name = ct.PortName
			port = ct.getPort()
		}
	}

	if name == "" {
		name = e.determineDefaultContainerPortName()
	}

	return &corev1.ContainerPort{
		Name:          name,
		ContainerPort: port,
		Protocol:      corev1.ProtocolTCP,
	}
}

// CapabilityPropertyKey returns the key or expand any variable provided in it. vars variable contain the
// possible dynamic values to use.
func CapabilityPropertyKey(camelPropertyKey string, vars map[string]string) string {
	if capabilityDynamicProperty.MatchString(camelPropertyKey) && vars != nil {
		match := capabilityDynamicProperty.FindStringSubmatch(camelPropertyKey)
		if len(match) < 2 {
			// Should not happen, but fallback to the key not expanded instead of panic if it comes to happen
			return camelPropertyKey
		}
		return strings.ReplaceAll(camelPropertyKey, match[1], vars[match[2]])
	}
	return camelPropertyKey
}

// ConsumeMeta is used to consume metadata information coming from Integration sources. If no sources available,
// would return false. When consuming from meta you should make sure that the configuration is stored in the
// status traits by setting each trait configuration when in "auto" mode.
// originalSourcesOnly flag indicates if you want to use only the sources provided originally to the Integration, otherwise
// it will consume all sources, also the one autogenerated by the operator.
func (e *Environment) ConsumeMeta(originalSourcesOnly bool, consumeMeta func(metadata.IntegrationMetadata) bool) (bool, error) {
	return e.consumeSourcesMeta(originalSourcesOnly, nil, consumeMeta)
}

// consumeSourcesMeta is used to consume both sources and metadata information coming from Integration sources.
// If no sources available would return false.
func (e *Environment) consumeSourcesMeta(
	originalSourcesOnly bool,
	consumeSources func(sources []v1.SourceSpec) bool,
	consumeMeta func(metadata.IntegrationMetadata) bool) (bool, error) {
	var sources []v1.SourceSpec
	var err error
	if sources, err = resolveIntegrationSources(e.Ctx, e.Client, e.Integration, originalSourcesOnly, e.Resources); err != nil {
		return false, err
	}
	if len(sources) < 1 {
		// No sources available
		return false, nil
	}
	if consumeSources != nil {
		consumeSources(sources)
	}
	if e.CamelCatalog == nil {
		return false, fmt.Errorf("cannot extract metadata from sources. Camel Catalog is null")
	}
	meta, err := metadata.ExtractAll(e.CamelCatalog, sources)
	if err != nil {
		return false, err
	}

	return consumeMeta(meta), nil
}

func (e *Environment) appendCloudPropertiesLocation(cloudPropertiesLocation string) {
	if e.ApplicationProperties["camel.main.cloud-properties-location"] == "" {
		e.ApplicationProperties["camel.main.cloud-properties-location"] = cloudPropertiesLocation
	} else {
		e.ApplicationProperties["camel.main.cloud-properties-location"] += "," + cloudPropertiesLocation
	}
}
