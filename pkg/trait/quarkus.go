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
	"fmt"
	"sort"
	"strings"

	"github.com/rs/xid"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

const (
	quarkusTraitID = "quarkus"

	QuarkusNativeDefaultBaseImageName = "quay.io/quarkus/quarkus-micro-image:2.0"
)

type quarkusPackageType string

const (
	fastJarPackageType       quarkusPackageType = "fast-jar"
	nativeSourcesPackageType quarkusPackageType = "native-sources"
)

var kitPriority = map[quarkusPackageType]string{
	fastJarPackageType:       "1000",
	nativeSourcesPackageType: "2000",
}

type quarkusTrait struct {
	BasePlatformTrait
	traitv1.QuarkusTrait `property:",squash"`
}
type languageSettings struct {
	// indicates whether the native mode is supported
	native bool
	// indicates whether the sources are required at build time for native compilation
	sourcesRequiredAtBuildTime bool
}

var (
	// settings for an unknown language.
	defaultSettings = languageSettings{false, false}
	// settings for languages supporting native mode for old catalogs.
	nativeSupportSettings = languageSettings{true, false}
)

// Retrieves the settings of the given language from the Camel catalog.
func getLanguageSettings(e *Environment, language v1.Language) languageSettings {
	if loader, ok := e.CamelCatalog.Loaders[string(language)]; ok {
		native, nExists := loader.Metadata["native"]
		if !nExists {
			log.Debug("The metadata 'native' is absent from the Camel catalog, the legacy language settings are applied")
			return getLegacyLanguageSettings(language)
		}
		sourcesRequiredAtBuildTime, sExists := loader.Metadata["sources-required-at-build-time"]
		return languageSettings{
			native:                     native == "true",
			sourcesRequiredAtBuildTime: sExists && sourcesRequiredAtBuildTime == "true",
		}
	}
	log.Debugf("No loader could be found for the language %q, the legacy language settings are applied", string(language))
	return getLegacyLanguageSettings(language)
}

// Provides the legacy settings of a given language.
func getLegacyLanguageSettings(language v1.Language) languageSettings {
	switch language {
	case v1.LanguageXML, v1.LanguageYaml, v1.LanguageKamelet:
		return nativeSupportSettings
	default:
		return defaultSettings
	}
}

func newQuarkusTrait() Trait {
	return &quarkusTrait{
		BasePlatformTrait: NewBasePlatformTrait(quarkusTraitID, 1700),
	}
}

// InfluencesKit overrides base class method.
func (t *quarkusTrait) InfluencesKit() bool {
	return true
}

// InfluencesBuild overrides base class method.
func (t *quarkusTrait) InfluencesBuild(this, prev map[string]interface{}) bool {
	return true
}

var _ ComparableTrait = &quarkusTrait{}

func (t *quarkusTrait) Matches(trait Trait) bool {
	qt, ok := trait.(*quarkusTrait)
	if !ok {
		return false
	}

	if len(t.Modes) == 0 && len(qt.Modes) != 0 && !qt.containsMode(traitv1.JvmQuarkusMode) {
		return false
	}

	for _, md := range t.Modes {
		if md == traitv1.JvmQuarkusMode && len(qt.Modes) == 0 {
			continue
		}
		if qt.containsMode(md) {
			continue
		}
		return false
	}

	return true
}

func (t *quarkusTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	condition := t.adaptDeprecatedFields()

	return e.IntegrationInPhase(v1.IntegrationPhaseBuildingKit) ||
			e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) ||
			e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) && e.IntegrationInRunningPhases(),
		condition, nil
}

func (t *quarkusTrait) adaptDeprecatedFields() *TraitCondition {
	if t.PackageTypes != nil {
		message := "The package-type parameter is deprecated and may be removed in future releases. Make sure to use mode parameter instead."
		t.L.Info(message)
		for _, pt := range t.PackageTypes {
			if pt == traitv1.NativePackageType {
				t.Modes = append(t.Modes, traitv1.NativeQuarkusMode)
				continue
			}
			if pt == traitv1.FastJarPackageType {
				t.Modes = append(t.Modes, traitv1.JvmQuarkusMode)
			}
		}
		return NewIntegrationCondition(v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, traitConfigurationReason, message)
	}

	return nil
}

func (t *quarkusTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseBuildingKit) {
		t.applyWhileBuildingKit(e)

		return nil
	}

	switch e.IntegrationKit.Status.Phase {
	case v1.IntegrationKitPhaseBuildSubmitted:
		if err := t.applyWhenBuildSubmitted(e); err != nil {
			return err
		}

	case v1.IntegrationKitPhaseReady:
		if err := t.applyWhenKitReady(e); err != nil {
			return err
		}
	}

	return nil
}

func (t *quarkusTrait) applyWhileBuildingKit(e *Environment) {
	if t.containsMode(traitv1.NativeQuarkusMode) {
		// Native compilation is only supported for a subset of languages,
		// so let's check for compatibility, and fail-fast the Integration,
		// to save compute resources and user time.
		if !t.validateNativeSupport(e) {
			// Let the calling controller handle the Integration update
			return
		}
	}

	switch len(t.Modes) {
	case 0:
		// Default behavior
		kit := t.newIntegrationKit(e, fastJarPackageType)
		e.IntegrationKits = append(e.IntegrationKits, *kit)
	case 1:
		kit := t.newIntegrationKit(e, packageType(t.Modes[0]))
		e.IntegrationKits = append(e.IntegrationKits, *kit)
	default:
		for _, md := range t.Modes {
			kit := t.newIntegrationKit(e, packageType(md))
			if kit.Spec.Traits.Quarkus == nil {
				kit.Spec.Traits.Quarkus = &traitv1.QuarkusTrait{}
			}
			kit.Spec.Traits.Quarkus.Modes = []traitv1.QuarkusMode{md}
			e.IntegrationKits = append(e.IntegrationKits, *kit)
		}
	}
}

func (t *quarkusTrait) validateNativeSupport(e *Environment) bool {
	for _, source := range e.Integration.Sources() {
		if language := source.InferLanguage(); !getLanguageSettings(e, language).native {
			t.L.ForIntegration(e.Integration).Infof("Integration %s/%s contains a %s source that cannot be compiled to native executable", e.Integration.Namespace, e.Integration.Name, language)
			e.Integration.Status.Phase = v1.IntegrationPhaseError
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionKitAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionUnsupportedLanguageReason,
				fmt.Sprintf("native compilation for language %q is not supported", language))

			return false
		}
	}

	return true
}

func (t *quarkusTrait) newIntegrationKit(e *Environment, packageType quarkusPackageType) *v1.IntegrationKit {
	integration := e.Integration
	kit := v1.NewIntegrationKit(integration.GetIntegrationKitNamespace(e.Platform), fmt.Sprintf("kit-%s", xid.New()))

	kit.Labels = map[string]string{
		v1.IntegrationKitTypeLabel:            v1.IntegrationKitTypePlatform,
		"camel.apache.org/runtime.version":    integration.Status.RuntimeVersion,
		"camel.apache.org/runtime.provider":   string(integration.Status.RuntimeProvider),
		v1.IntegrationKitLayoutLabel:          string(packageType),
		v1.IntegrationKitPriorityLabel:        kitPriority[packageType],
		kubernetes.CamelCreatorLabelKind:      v1.IntegrationKind,
		kubernetes.CamelCreatorLabelName:      integration.Name,
		kubernetes.CamelCreatorLabelNamespace: integration.Namespace,
		kubernetes.CamelCreatorLabelVersion:   integration.ResourceVersion,
	}

	if v, ok := integration.Annotations[v1.PlatformSelectorAnnotation]; ok {
		v1.SetAnnotation(&kit.ObjectMeta, v1.PlatformSelectorAnnotation, v)
	}

	for k, v := range integration.Annotations {
		if strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
			v1.SetAnnotation(&kit.ObjectMeta, k, v)
		}
	}

	operatorID := defaults.OperatorID()
	if operatorID != "" {
		kit.SetOperatorID(operatorID)
	}

	kit.Spec = v1.IntegrationKitSpec{
		Dependencies: e.Integration.Status.Dependencies,
		Repositories: e.Integration.Spec.Repositories,
		Traits:       propagateKitTraits(e),
	}

	if packageType == nativeSourcesPackageType {
		kit.Spec.Sources = propagateSourcesRequiredAtBuildTime(e)
	}
	return kit
}

func propagateKitTraits(e *Environment) v1.IntegrationKitTraits {
	traits := e.Integration.Spec.Traits
	kitTraits := v1.IntegrationKitTraits{
		Builder:  traits.Builder.DeepCopy(),
		Camel:    traits.Camel.DeepCopy(),
		Quarkus:  traits.Quarkus.DeepCopy(),
		Registry: traits.Registry.DeepCopy(),
	}

	// propagate addons that influence kits too
	if len(traits.Addons) > 0 {
		kitTraits.Addons = make(map[string]v1.AddonTrait)
		for id, addon := range traits.Addons {
			if t := e.Catalog.GetTrait(id); t != nil && t.InfluencesKit() {
				kitTraits.Addons[id] = *addon.DeepCopy()
			}
		}
	}

	return kitTraits
}

func (t *quarkusTrait) applyWhenBuildSubmitted(e *Environment) error {
	buildTask := getBuilderTask(e.Pipeline)
	if buildTask == nil {
		return fmt.Errorf("unable to find builder task: %s", e.Integration.Name)
	}
	packageTask := getPackageTask(e.Pipeline)
	if packageTask == nil {
		return fmt.Errorf("unable to find package task: %s", e.Integration.Name)
	}

	buildSteps, err := builder.StepsFrom(buildTask.Steps...)
	if err != nil {
		return err
	}
	buildSteps = append(buildSteps, builder.Quarkus.CommonSteps...)

	packageSteps, err := builder.StepsFrom(packageTask.Steps...)
	if err != nil {
		return err
	}

	if buildTask.Maven.Properties == nil {
		buildTask.Maven.Properties = make(map[string]string)
	}

	native, err := t.isNativeKit(e)
	if err != nil {
		return err
	}

	// The LoadCamelQuarkusCatalog is required to have catalog information available by the builder
	packageSteps = append(packageSteps, builder.Quarkus.LoadCamelQuarkusCatalog)

	if native {
		if nativePackageType := builder.QuarkusRuntimeSupport(e.CamelCatalog.GetCamelQuarkusVersion()).NativeMavenProperty(); nativePackageType != "" {
			buildTask.Maven.Properties["quarkus.package.type"] = nativePackageType
			if t.NativeBaseImage == "" {
				packageTask.BaseImage = QuarkusNativeDefaultBaseImageName
			} else {
				packageTask.BaseImage = t.NativeBaseImage
			}
		}
		if len(e.IntegrationKit.Spec.Sources) > 0 {
			buildTask.Sources = e.IntegrationKit.Spec.Sources
			buildSteps = append(buildSteps, builder.Quarkus.PrepareProjectWithSources)
		}
		packageSteps = append(packageSteps, builder.Image.NativeImageContext)
		// Create the dockerfile, regardless it's later used or not by the publish strategy
		packageSteps = append(packageSteps, builder.Image.ExecutableDockerfile)
	} else {
		// Default, if nothing is specified
		buildTask.Maven.Properties["quarkus.package.type"] = string(fastJarPackageType)
		packageSteps = append(packageSteps, builder.Quarkus.ComputeQuarkusDependencies)
		if t.isIncrementalImageBuild(e) {
			packageSteps = append(packageSteps, builder.Image.IncrementalImageContext)
		} else {
			packageSteps = append(packageSteps, builder.Image.StandardImageContext)
		}
		// Create the dockerfile, regardless it's later used or not by the publish strategy
		packageSteps = append(packageSteps, builder.Image.JvmDockerfile)
	}

	// Sort steps by phase
	sort.SliceStable(buildSteps, func(i, j int) bool {
		return buildSteps[i].Phase() < buildSteps[j].Phase()
	})
	sort.SliceStable(packageSteps, func(i, j int) bool {
		return packageSteps[i].Phase() < packageSteps[j].Phase()
	})

	buildTask.Steps = builder.StepIDsFor(buildSteps...)
	packageTask.Steps = builder.StepIDsFor(packageSteps...)

	return nil
}

func (t *quarkusTrait) isNativeKit(e *Environment) (bool, error) {
	switch modes := t.Modes; len(modes) {
	case 0:
		return false, nil
	case 1:
		return modes[0] == traitv1.NativeQuarkusMode, nil
	default:
		return false, fmt.Errorf("kit %q has more than one package type", e.IntegrationKit.Name)
	}
}

func (t *quarkusTrait) isIncrementalImageBuild(e *Environment) bool {
	// We need to get this information from the builder trait
	if trait := e.Catalog.GetTrait(builderTraitID); trait != nil {
		builder, ok := trait.(*builderTrait)
		return ok && pointer.BoolDeref(builder.IncrementalImageBuild, true)
	}

	// Default always to true for performance reasons
	return true
}

func (t *quarkusTrait) applyWhenKitReady(e *Environment) error {
	if e.IntegrationInRunningPhases() && t.isNativeIntegration(e) {
		container := e.GetIntegrationContainer()
		if container == nil {
			return fmt.Errorf("unable to find integration container: %s", e.Integration.Name)
		}

		container.Command = []string{"./camel-k-integration-" + defaults.Version + "-runner"}
		container.WorkingDir = builder.DeploymentDir
	}

	return nil
}

func (t *quarkusTrait) isNativeIntegration(e *Environment) bool {
	// The current IntegrationKit determines the Integration runtime type
	return e.IntegrationKit.Labels[v1.IntegrationKitLayoutLabel] == v1.IntegrationKitLayoutNativeSources
}

// Indicates whether the given source code is embedded into the final binary.
func (t *quarkusTrait) isEmbedded(e *Environment, source v1.SourceSpec) bool {
	if e.IntegrationInRunningPhases() {
		return e.IntegrationKit != nil && t.isNativeIntegration(e) && sourcesRequiredAtBuildTime(e, source)
	} else if e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) {
		native, _ := t.isNativeKit(e)
		return native && sourcesRequiredAtBuildTime(e, source)
	}
	return false
}

func (t *quarkusTrait) containsMode(m traitv1.QuarkusMode) bool {
	for _, mode := range t.Modes {
		if mode == m {
			return true
		}
	}
	return false
}

func packageType(mode traitv1.QuarkusMode) quarkusPackageType {
	if mode == traitv1.NativeQuarkusMode {
		return nativeSourcesPackageType
	}
	if mode == traitv1.JvmQuarkusMode {
		return fastJarPackageType
	}

	return ""
}

// Indicates whether the given source file is required at build time for native compilation.
func sourcesRequiredAtBuildTime(e *Environment, source v1.SourceSpec) bool {
	settings := getLanguageSettings(e, source.InferLanguage())
	return settings.native && settings.sourcesRequiredAtBuildTime
}

// Propagates the sources that are required at build time for native compilation.
func propagateSourcesRequiredAtBuildTime(e *Environment) []v1.SourceSpec {
	array := make([]v1.SourceSpec, 0)
	for _, source := range e.Integration.Sources() {
		if sourcesRequiredAtBuildTime(e, source) {
			array = append(array, source)
		}
	}
	return array
}
