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
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/kamelet/repository"
	"github.com/apache/camel-k/v2/pkg/metadata"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/dsl"
	"github.com/apache/camel-k/v2/pkg/util/source"
)

const (
	kameletsTraitID    = "kamelets"
	kameletsTraitOrder = 450

	contentKey                  = "content"
	KameletLocationProperty     = "camel.component.kamelet.location"
	kameletLabel                = "camel.apache.org/kamelet"
	kameletConfigurationLabel   = "camel.apache.org/kamelet.configuration"
	kameletMountPointAnnotation = "camel.apache.org/kamelet.mount-point"
)

var kameletVersionProperty = fmt.Sprintf("?%s=", v1.KameletVersionProperty)

type kameletsTrait struct {
	BaseTrait
	traitv1.KameletsTrait `property:",squash"`
}

func newKameletsTrait() Trait {
	return &kameletsTrait{
		BaseTrait: NewBaseTrait(kameletsTraitID, kameletsTraitOrder),
	}
}

func (t *kameletsTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationConditionUserDisabled("Kamelets"), nil
	}
	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}
	if t.MountPoint == "" {
		t.MountPoint = filepath.Join(camel.BasePath, "kamelets")
	}
	if ptr.Deref(t.Auto, true) {
		var kamelets []string
		_, err := e.ConsumeMeta(func(meta metadata.IntegrationMetadata) bool {
			util.StringSliceUniqueConcat(&kamelets, meta.Kamelets)
			return true
		})
		if err != nil {
			return false, nil, err
		}
		// Check if a Kamelet is configured as default error handler URI
		defaultErrorHandlerURI := e.Integration.Spec.GetConfigurationProperty(v1.ErrorHandlerAppPropertiesPrefix + ".deadLetterUri")
		if defaultErrorHandlerURI != "" && strings.HasPrefix(defaultErrorHandlerURI, "kamelet:") {
			kamelets = append(kamelets, source.ExtractKamelet(defaultErrorHandlerURI))
		}
		if len(kamelets) > 0 {
			sort.Strings(kamelets)
			t.List = strings.Join(kamelets, ",")
		}
	}

	return len(t.getKameletKeys(false)) > 0, nil, nil
}

func (t *kameletsTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases() {
		if err := t.addKamelets(e); err != nil {
			return err
		}
	}
	return nil
}

// collectKamelets load a Kamelet specification setting the specific version specification.
func (t *kameletsTrait) collectKamelets(e *Environment) (map[string]*v1.Kamelet, error) {
	repo, err := repository.NewForPlatform(e.Ctx, e.Client, e.Platform, e.Integration.Namespace, platform.GetOperatorNamespace())
	if err != nil {
		return nil, err
	}

	kamelets := make(map[string]*v1.Kamelet)
	missingKamelets := make([]string, 0)
	availableKamelets := make([]string, 0)
	bundledKamelets := make([]string, 0)

	for _, kml := range strings.Split(t.List, ",") {
		name := getKameletKey(kml, false)
		if !v1.ValidKameletName(name) {
			// Skip kamelet sink and source id
			continue
		}
		kamelet, err := repo.Get(e.Ctx, name)
		if err != nil {
			return nil, err
		}
		if kamelet == nil {
			missingKamelets = append(missingKamelets, name)
			continue
		} else {
			availableKamelets = append(availableKamelets, name)
		}
		if kamelet.IsBundled() {
			bundledKamelets = append(bundledKamelets, name)
		}
		// We control which version to use (if any is specified)
		clonedKamelet, err := kamelet.CloneWithVersion(getKameletVersion(kml))
		if err != nil {
			return nil, err
		}
		kamelets[clonedKamelet.Name] = clonedKamelet
	}

	sort.Strings(availableKamelets)
	sort.Strings(missingKamelets)
	sort.Strings(bundledKamelets)

	if len(missingKamelets) > 0 {
		message := fmt.Sprintf("kamelets [%s] not found in %s repositories",
			strings.Join(missingKamelets, ","),
			repo.String())
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionKameletsAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKameletsAvailableReason,
			message,
		)

		return nil, errors.New(message)
	}

	// TODO:
	// We list the Kamelets coming from a bundle. We want to warn the user
	// that in the future we'll use the specification coming from the dependency runtime
	// instead of using the one installed in the cluster.
	// It may be a good idea in the future to let the user specify the catalog dependency to use
	// in order to override the one coming from Apache catalog
	if len(bundledKamelets) > 0 {
		message := fmt.Sprintf("using bundled kamelets [%s]: make sure the Kamelet specifications is compatible with this Integration runtime."+
			" This feature is deprecated as in the future we will use directly the specification coming from the Kamelet catalog dependency jar.",
			strings.Join(bundledKamelets, ","))
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionType("KameletsDeprecationNotice"),
			corev1.ConditionTrue,
			"KameletsDeprecationNotice",
			message,
		)
	}

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionKameletsAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionKameletsAvailableReason,
		fmt.Sprintf("kamelets [%s] found in %s repositories", strings.Join(availableKamelets, ","), repo.String()),
	)

	return kamelets, nil
}

func (t *kameletsTrait) addKamelets(e *Environment) error {
	if len(t.getKameletKeys(false)) == 0 {
		return nil
	}
	kamelets, err := t.collectKamelets(e)
	if err != nil {
		return err
	}
	kb := newKameletBundle()
	for _, kamelet := range kamelets {
		if err := t.addKameletAsSource(e, kamelet); err != nil {
			return err
		}
		// Adding dependencies from Kamelets
		util.StringSliceUniqueConcat(&e.Integration.Status.Dependencies, kamelet.Spec.Dependencies)
		// Add to Kamelet bundle configmap
		kb.add(kamelet)
	}
	bundleConfigmaps, err := kb.toConfigmaps(e.Integration.Name, e.Integration.Namespace)
	if err != nil {
		return err
	}
	// set kamelets runtime location
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = map[string]string{}
	}
	for _, cm := range bundleConfigmaps {
		kameletMountPoint := fmt.Sprintf("%s/%s", t.MountPoint, cm.Name)
		cm.Annotations[kameletMountPointAnnotation] = kameletMountPoint
		e.Resources.Add(cm)
		if e.ApplicationProperties[KameletLocationProperty] == "" {
			e.ApplicationProperties[KameletLocationProperty] = fmt.Sprintf("file:%s", kameletMountPoint)
		} else {
			e.ApplicationProperties[KameletLocationProperty] += fmt.Sprintf(",file:%s", kameletMountPoint)
		}
	}
	e.ApplicationProperties[KameletLocationProperty] += ",classpath:/kamelets"
	// resort dependencies
	sort.Strings(e.Integration.Status.Dependencies)

	return nil
}

// This func will add a Kamelet as a generated Integration source. The source included here is going to be used in order to parse the Kamelet
// for any component or capability (ie, rest) which is included in the Kamelet spec itself. However, the generated source is marked as coming `FromKamelet`.
// When mounting the sources, these generated sources won't be mounted as sources but as Kamelet instead.
func (t *kameletsTrait) addKameletAsSource(e *Environment, kamelet *v1.Kamelet) error {
	sources := make([]v1.SourceSpec, 0)

	if kamelet.Spec.Template != nil {
		flowData, err := dsl.TemplateToYamlDSL(*kamelet.Spec.Template, kamelet.Name)
		if err != nil {
			return err
		}
		flowSource := v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:    fmt.Sprintf("%s.yaml", kamelet.Name),
				Content: string(flowData),
			},
			Language: v1.LanguageYaml,
		}
		flowSource, err = integrationSourceFromKameletSource(e, kamelet, flowSource, fmt.Sprintf("%s-kamelet-%s-template", e.Integration.Name, kamelet.Name))
		if err != nil {
			return err
		}
		sources = append(sources, flowSource)
	}

	for idx, s := range kamelet.Spec.Sources {
		intSource, err := integrationSourceFromKameletSource(e, kamelet, s, fmt.Sprintf("%s-kamelet-%s-%03d", e.Integration.Name, kamelet.Name, idx))
		if err != nil {
			return err
		}
		sources = append(sources, intSource)
	}

	for _, source := range sources {
		replaced := false
		for idx, existing := range e.Integration.Status.GeneratedSources {
			if existing.Name == source.Name {
				replaced = true
				e.Integration.Status.GeneratedSources[idx] = source
			}
		}
		if !replaced {
			e.Integration.Status.GeneratedSources = append(e.Integration.Status.GeneratedSources, source)
		}
	}

	return nil
}

func (t *kameletsTrait) getKameletKeys(withVersion bool) []string {
	answer := make([]string, 0)
	for _, item := range strings.Split(t.List, ",") {
		i := getKameletKey(item, withVersion)
		if i != "" && v1.ValidKameletName(i) {
			util.StringSliceUniqueAdd(&answer, i)
		}
	}
	sort.Strings(answer)
	return answer
}

func getKameletKey(item string, withVersion bool) string {
	i := strings.Trim(item, " \t\"")
	if strings.Contains(i, "/") {
		i = strings.SplitN(i, "/", 2)[0]
	}
	if strings.Contains(i, kameletVersionProperty) {
		versionedKamelet := strings.SplitN(i, kameletVersionProperty, 2)
		if withVersion {
			i = fmt.Sprintf("%s-%s", versionedKamelet[0], versionedKamelet[1])
		} else {
			i = versionedKamelet[0]
		}
	}
	return i
}

func getKameletVersion(item string) string {
	if strings.Contains(item, fmt.Sprintf("?%s=", v1.KameletVersionProperty)) {
		versionedKamelet := strings.SplitN(item, kameletVersionProperty, 2)
		return versionedKamelet[1]
	}
	return ""
}

func integrationSourceFromKameletSource(e *Environment, kamelet *v1.Kamelet, source v1.SourceSpec, name string) (v1.SourceSpec, error) {
	if source.Type == v1.SourceTypeTemplate {
		// Kamelets must be named "<kamelet-name>.extension"
		language := source.InferLanguage()
		source.Name = fmt.Sprintf("%s.%s", kamelet.Name, string(language))
	}

	source.FromKamelet = true

	if source.DataSpec.ContentRef != "" {
		return source, nil
	}

	// Create configmaps to avoid storing kamelet definitions in the integration CR
	// Compute the input digest and store it along with the configmap
	hash, err := digest.ComputeForSource(source)
	if err != nil {
		return v1.SourceSpec{}, err
	}
	cm := initializeConfigmapKameletSource(source, hash, name, e.Integration.Namespace, e.Integration.Name, kamelet.Name)
	e.Resources.Add(&cm)

	target := source.DeepCopy()
	target.Content = ""
	target.ContentRef = name
	target.ContentKey = contentKey
	return *target, nil
}

func initializeConfigmapKameletSource(source v1.SourceSpec, hash, name, namespace, itName, kamName string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": itName,
				"camel.apache.org/kamelet":     kamName,
			},
			Annotations: map[string]string{
				sourceLanguageAnnotation:            string(source.Language),
				sourceNameAnnotation:                name,
				sourceCompressionAnnotation:         strconv.FormatBool(source.Compression),
				"camel.apache.org/source.generated": boolean.TrueString,
				"camel.apache.org/source.type":      string(source.Type),
				"camel.apache.org/source.digest":    hash,
			},
		},
		Data: map[string]string{
			contentKey: source.Content,
		},
	}
}
