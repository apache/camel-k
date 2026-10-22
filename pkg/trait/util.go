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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/property"
	"github.com/apache/camel-k/v2/pkg/util/sets"
)

type Options map[string]map[string]any

func (u Options) Get(id string) (map[string]any, bool) {
	if t, ok := u[id]; ok {
		return t, true
	}

	return nil, false
}

var exactVersionRegexp = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)([\w-.]*)$`)

// getIntegrationKit retrieves the kit set on the integration.
func getIntegrationKit(ctx context.Context, c client.Client, integration *v1.Integration) (*v1.IntegrationKit, error) {
	if integration.Status.IntegrationKit == nil {
		return nil, nil
	}

	return kubernetes.GetIntegrationKit(ctx, c, integration.Status.IntegrationKit.Name, integration.Status.IntegrationKit.Namespace)
}

func collectConfigurationPairs(configurationType string, configurable ...v1.Configurable) []variable {
	result := make([]variable, 0)

	for _, c := range configurable {
		if c == nil || reflect.ValueOf(c).IsNil() {
			continue
		}

		entries := c.Configurations()
		if entries == nil {
			continue
		}

		for _, entry := range entries {
			if entry.Type == configurationType {
				k, v := property.SplitPropertyFileEntry(entry.Value)
				if k == "" {
					continue
				}

				ok := false
				for i, variable := range result {
					if variable.Name == k {
						result[i].Value = v
						ok = true

						break
					}
				}
				if !ok {
					result = append(result, variable{Name: k, Value: v})
				}
			}
		}
	}

	return result
}

var keyValuePairRegexp = regexp.MustCompile(`^(\w+)=(.+)$`)

func keyValuePairArrayAsStringMap(pairs []string) (map[string]string, error) {
	m := make(map[string]string)

	for _, pair := range pairs {
		if match := keyValuePairRegexp.FindStringSubmatch(pair); match != nil {
			m[match[1]] = match[2]
		} else {
			return nil, fmt.Errorf("unable to parse key/value pair: %s", pair)
		}
	}

	return m, nil
}

// filterTransferableAnnotations returns a map containing annotations that are meaningful for being transferred to child resources.
func filterTransferableAnnotations(annotations map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range annotations {
		if strings.HasPrefix(k, "kubectl.kubernetes.io") {
			// filter out kubectl annotations
			continue
		}
		res[k] = v
	}

	return res
}

// ExtractSourceLoaderDependencies extracts dependencies from source.
func ExtractSourceLoaderDependencies(source v1.SourceSpec, catalog *camel.RuntimeCatalog) *sets.Set {
	dependencies := sets.NewSet()
	// Add loader dependencies
	lang := source.InferLanguage()
	for loader, v := range catalog.Loaders {
		// add loader specific dependencies
		if source.Loader != "" && source.Loader == loader {
			dependencies.Add(v.GetDependencyID())

			for _, d := range v.Dependencies {
				dependencies.Add(d.GetDependencyID())
			}
		} else if source.Loader == "" {
			// add language specific dependencies
			if util.StringSliceExists(v.Languages, string(lang)) {
				dependencies.Add(v.GetDependencyID())

				for _, d := range v.Dependencies {
					dependencies.Add(d.GetDependencyID())
				}
			}
		}
	}

	return dependencies
}

// AssertTraitsType asserts that traits is either v1.Traits or v1.IntegrationKitTraits.
// This function is provided because Go doesn't have Either nor union types.
func AssertTraitsType(traits any) error {
	_, ok1 := traits.(v1.Traits)
	_, ok2 := traits.(v1.IntegrationKitTraits)
	if !ok1 && !ok2 {
		return errors.New("traits must be either v1.Traits or v1.IntegrationKitTraits")
	}

	return nil
}

// ToTraitMap accepts either v1.Traits or v1.IntegrationKitTraits and converts it to a map of traits.
func ToTraitMap(traits any) (Options, error) {
	if err := AssertTraitsType(traits); err != nil {
		return nil, err
	}

	data, err := json.Marshal(traits)
	if err != nil {
		return nil, err
	}
	traitMap := make(Options)
	if err = json.Unmarshal(data, &traitMap); err != nil {
		return nil, err
	}

	return traitMap, nil
}

// MigrateLegacyConfiguration moves up the legacy configuration in a trait to the new top-level properties.
// Values of the new properties always take precedence over the ones from the legacy configuration
// with the same property names.
func MigrateLegacyConfiguration(trait map[string]any) error {
	if trait["configuration"] == nil {
		return nil
	}

	if config, ok := trait["configuration"].(map[string]any); ok {
		// For traits that had the same property name "configuration",
		// the property needs to be renamed to "config" to avoid naming conflicts
		// (e.g. Knative trait).
		if config["configuration"] != nil {
			config["config"] = config["configuration"]
			delete(config, "configuration")
		}

		for k, v := range config {
			if trait[k] == nil {
				trait[k] = v
			}
		}
		delete(trait, "configuration")
	} else {
		return fmt.Errorf(`unexpected type for "configuration" field: %v`, reflect.TypeOf(trait["configuration"]))
	}

	return nil
}

// ToTrait unmarshals a map configuration to a target trait.
func ToTrait(trait map[string]any, target any) error {
	data, err := json.Marshal(trait)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &target)
	if err != nil {
		return err
	}

	return nil
}

func getBuilderTask(tasks []v1.Task) *v1.BuilderTask {
	for i, task := range tasks {
		if task.Builder != nil {
			return tasks[i].Builder
		}
	}

	return nil
}

func getPackageTask(tasks []v1.Task) *v1.BuilderTask {
	for i, task := range tasks {
		if task.Package != nil {
			return tasks[i].Package
		}
	}

	return nil
}

// Equals return if traits are the same.
func Equals(i1 Options, i2 Options) bool {
	return reflect.DeepEqual(i1, i2)
}

// IntegrationsHaveSameTraits return if traits are the same.
func IntegrationsHaveSameTraits(i1 *v1.Integration, i2 *v1.Integration) (bool, error) {
	c1, err := NewSpecTraitsOptionsForIntegration(i1)
	if err != nil {
		return false, err
	}
	c2, err := NewSpecTraitsOptionsForIntegration(i2)
	if err != nil {
		return false, err
	}

	return Equals(c1, c2), nil
}

// PipesHaveSameTraits return if traits are the same.
func PipesHaveSameTraits(i1 *v1.Pipe, i2 *v1.Pipe) (bool, error) {
	c1, err := NewTraitsOptionsForPipe(i1)
	if err != nil {
		return false, err
	}
	c2, err := NewTraitsOptionsForPipe(i2)
	if err != nil {
		return false, err
	}

	return Equals(c1, c2), nil
}

// IntegrationAndPipeSameTraits return if traits are the same.
// The comparison is done for the subset of traits defines on the binding as during the trait processing,
// some traits may be added to the Integration i.e. knative configuration in case of sink binding.
func IntegrationAndPipeSameTraits(i1 *v1.Integration, i2 *v1.Pipe) (bool, error) {
	itOpts, err := NewSpecTraitsOptionsForIntegration(i1)
	if err != nil {
		return false, err
	}
	klbOpts, err := NewTraitsOptionsForPipe(i2)
	if err != nil {
		return false, err
	}
	toCompare := make(Options)
	for k := range klbOpts {
		if v, ok := itOpts[k]; ok {
			toCompare[k] = v
		}
	}

	return Equals(klbOpts, toCompare), nil
}

// NewSpecTraitsOptionsForIntegrationAndPlatform will merge traits giving priority to Integration, Profile and Platform respectively.
//
//nolint:staticcheck
func NewSpecTraitsOptionsForIntegrationAndPlatform(i *v1.Integration, itp *v1.IntegrationProfile, pl *v1.IntegrationPlatform) (Options, error) {
	mergedTraits := v1.Traits{}
	itpTraits := v1.Traits{}
	if pl != nil {
		mergedTraits = pl.Status.Traits
	}
	if itp != nil {
		itpTraits = itp.Spec.Traits
	}
	if err := mergedTraits.Merge(itpTraits); err != nil {
		return nil, err
	}
	if err := mergedTraits.Merge(i.Spec.Traits); err != nil {
		return nil, err
	}

	return ToTraitMap(mergedTraits)
}

func NewSpecTraitsOptionsForIntegration(it *v1.Integration) (Options, error) {
	return ToTraitMap(it.Spec.Traits)
}

func NewSpecTraitsOptionsForIntegrationKit(ik *v1.IntegrationKit) (Options, error) {
	return ToTraitMap(ik.Spec.Traits)
}

func NewTraitsOptionsForPipe(pipe *v1.Pipe) (Options, error) {
	options := Options{}

	if pipe.Spec.Traits != nil {
		var err error
		options, err = ToTraitMap(*pipe.Spec.Traits)
		if err != nil {
			return nil, err
		}
	}

	return options, nil
}

// HasMatchingTraits verifies if two traits options match.
func HasMatchingTraits(traitMap Options, kitTraitMap Options) (bool, error) {
	catalog := NewCatalog(nil)

	for _, t := range catalog.AllTraits() {
		if t == nil || !t.InfluencesKit() {
			// We don't store the trait configuration if the trait cannot influence the kit behavior
			continue
		}
		id := string(t.ID())
		it, _ := traitMap.Get(id)
		kt, _ := kitTraitMap.Get(id)
		if ct, ok := t.(ComparableTrait); ok {
			// if it's match trait use its matches method to determine the match
			if match, err := matchesComparableTrait(ct, it, kt); !match || err != nil {
				return false, err
			}
		} else {
			if !matchesTrait(it, kt) {
				return false, nil
			}
		}
	}

	return true, nil
}

func matchesComparableTrait(ct ComparableTrait, it map[string]any, kt map[string]any) (bool, error) {
	t1 := reflect.New(reflect.TypeOf(ct).Elem()).Interface()
	if err := ToTrait(it, &t1); err != nil {
		return false, err
	}
	t2 := reflect.New(reflect.TypeOf(ct).Elem()).Interface()
	if err := ToTrait(kt, &t2); err != nil {
		return false, err
	}
	ct2, ok := t2.(ComparableTrait)
	if !ok {
		return false, fmt.Errorf("type assertion failed: %v", t2)
	}
	tt1, ok := t1.(Trait)
	if !ok {
		return false, fmt.Errorf("type assertion failed: %v", t1)
	}

	return ct2.Matches(tt1), nil
}

func matchesTrait(it map[string]any, kt map[string]any) bool {
	// perform exact match on the two trait maps
	return reflect.DeepEqual(it, kt)
}
