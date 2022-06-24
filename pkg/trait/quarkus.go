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

	"github.com/rs/xid"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

const (
	quarkusTraitID = "quarkus"
)

var kitPriority = map[traitv1.QuarkusPackageType]string{
	traitv1.FastJarPackageType: "1000",
	traitv1.NativePackageType:  "2000",
}

type quarkusTrait struct {
	BaseTrait
	traitv1.QuarkusTrait `property:",squash"`
}

func newQuarkusTrait() Trait {
	return &quarkusTrait{
		BaseTrait: NewBaseTrait(quarkusTraitID, 1700),
	}
}

// IsPlatformTrait overrides base class method.
func (t *quarkusTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method.
func (t *quarkusTrait) InfluencesKit() bool {
	return true
}

var _ ComparableTrait = &quarkusTrait{}

func (t *quarkusTrait) Matches(trait Trait) bool {
	qt, ok := trait.(*quarkusTrait)
	if !ok {
		return false
	}

	if pointer.BoolDeref(t.Enabled, true) && !pointer.BoolDeref(qt.Enabled, true) {
		return false
	}

	if len(t.PackageTypes) == 0 && len(qt.PackageTypes) != 0 && !containsPackageType(qt.PackageTypes, traitv1.FastJarPackageType) {
		return false
	}

	for _, pt := range t.PackageTypes {
		if pt == traitv1.FastJarPackageType && len(qt.PackageTypes) == 0 {
			continue
		}
		if containsPackageType(qt.PackageTypes, pt) {
			continue
		}
		return false
	}

	return true
}

func (t *quarkusTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseBuildingKit) ||
			e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) ||
			e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) && e.IntegrationInRunningPhases(),
		nil
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
	if containsPackageType(t.PackageTypes, traitv1.NativePackageType) {
		// Native compilation is only supported for a subset of languages,
		// so let's check for compatibility, and fail-fast the Integration,
		// to save compute resources and user time.
		if !t.validateNativeSupport(e) {
			// Let the calling controller handle the Integration update
			return
		}
	}

	switch len(t.PackageTypes) {
	case 0:
		kit := t.newIntegrationKit(e, traitv1.FastJarPackageType)
		e.IntegrationKits = append(e.IntegrationKits, *kit)

	case 1:
		kit := t.newIntegrationKit(e, t.PackageTypes[0])
		e.IntegrationKits = append(e.IntegrationKits, *kit)

	default:
		for _, pt := range t.PackageTypes {
			packageType := pt
			kit := t.newIntegrationKit(e, packageType)
			if kit.Spec.Traits.Quarkus == nil {
				kit.Spec.Traits.Quarkus = &traitv1.QuarkusTrait{}
			}
			kit.Spec.Traits.Quarkus.PackageTypes = []traitv1.QuarkusPackageType{packageType}
			e.IntegrationKits = append(e.IntegrationKits, *kit)
		}
	}
}

func (t *quarkusTrait) validateNativeSupport(e *Environment) bool {
	for _, source := range e.Integration.Sources() {
		if language := source.InferLanguage(); language != v1.LanguageKamelet &&
			language != v1.LanguageYaml &&
			language != v1.LanguageXML {
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

func (t *quarkusTrait) newIntegrationKit(e *Environment, packageType traitv1.QuarkusPackageType) *v1.IntegrationKit {
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
	operatorID := defaults.OperatorID()
	if operatorID != "" {
		kit.SetOperatorID(operatorID)
	}

	kit.Spec = v1.IntegrationKitSpec{
		Dependencies: e.Integration.Status.Dependencies,
		Repositories: e.Integration.Spec.Repositories,
		Traits:       propagateKitTraits(e),
	}

	return kit
}

func propagateKitTraits(e *Environment) v1.IntegrationKitTraits {
	traits := e.Integration.Spec.Traits
	kitTraits := v1.IntegrationKitTraits{
		Builder:  traits.Builder.DeepCopy(),
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
	build := getBuilderTask(e.BuildTasks)
	if build == nil {
		return fmt.Errorf("unable to find builder task: %s", e.Integration.Name)
	}

	if build.Maven.Properties == nil {
		build.Maven.Properties = make(map[string]string)
	}

	steps, err := builder.StepsFrom(build.Steps...)
	if err != nil {
		return err
	}

	steps = append(steps, builder.Quarkus.CommonSteps...)

	native, err := t.isNativeKit(e)
	if err != nil {
		return err
	}

	if native {
		build.Maven.Properties["quarkus.package.type"] = string(traitv1.NativePackageType)
		steps = append(steps, builder.Image.NativeImageContext)
		// Spectrum does not rely on Dockerfile to assemble the image
		if e.Platform.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategySpectrum {
			steps = append(steps, builder.Image.ExecutableDockerfile)
		}
	} else {
		build.Maven.Properties["quarkus.package.type"] = string(traitv1.FastJarPackageType)
		steps = append(steps, builder.Quarkus.ComputeQuarkusDependencies, builder.Image.IncrementalImageContext)
		// Spectrum does not rely on Dockerfile to assemble the image
		if e.Platform.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategySpectrum {
			steps = append(steps, builder.Image.JvmDockerfile)
		}
	}

	// Sort steps by phase
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].Phase() < steps[j].Phase()
	})

	build.Steps = builder.StepIDsFor(steps...)

	return nil
}

func (t *quarkusTrait) isNativeKit(e *Environment) (bool, error) {
	switch types := t.PackageTypes; len(types) {
	case 0:
		return false, nil
	case 1:
		return types[0] == traitv1.NativePackageType, nil
	default:
		return false, fmt.Errorf("kit %q has more than one package type", e.IntegrationKit.Name)
	}
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
	return e.IntegrationKit.Labels[v1.IntegrationKitLayoutLabel] == v1.IntegrationKitLayoutNative
}

func containsPackageType(types []traitv1.QuarkusPackageType, t traitv1.QuarkusPackageType) bool {
	for _, ti := range types {
		if t == ti {
			return true
		}
	}
	return false
}
