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
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	user "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/property"
)

type Options map[string]map[string]interface{}

func (u Options) Get(id string) (map[string]interface{}, bool) {
	if t, ok := u[id]; ok {
		return t, true
	}

	if addons, ok := u["addons"]; ok {
		if addon, ok := addons[id]; ok {
			if t, ok := addon.(map[string]interface{}); ok {
				return t, true
			}
		}
	}

	return nil, false
}

var exactVersionRegexp = regexp.MustCompile(`^(\d+)\.(\d+)\.([\w-.]+)$`)

// getIntegrationKit retrieves the kit set on the integration.
func getIntegrationKit(ctx context.Context, c client.Client, integration *v1.Integration) (*v1.IntegrationKit, error) {
	if integration.Status.IntegrationKit == nil {
		return nil, nil
	}
	kit := v1.NewIntegrationKit(integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name)
	err := c.Get(ctx, ctrl.ObjectKeyFromObject(kit), kit)
	return kit, err
}

func collectConfigurationValues(configurationType string, configurable ...v1.Configurable) []string {
	result := strset.New()

	for _, c := range configurable {
		c := c

		if c == nil || reflect.ValueOf(c).IsNil() {
			continue
		}

		entries := c.Configurations()
		if entries == nil {
			continue
		}

		for _, entry := range entries {
			if entry.Type == configurationType {
				result.Add(entry.Value)
			}
		}
	}

	s := result.List()
	sort.Strings(s)
	return s
}

func collectConfigurations(configurationType string, configurable ...v1.Configurable) []map[string]string {
	var result []map[string]string

	for _, c := range configurable {
		c := c

		if c == nil || reflect.ValueOf(c).IsNil() {
			continue
		}

		entries := c.Configurations()
		if entries == nil {
			continue
		}

		// nolint: staticcheck
		for _, entry := range entries {
			if entry.Type == configurationType {
				item := make(map[string]string)
				item["value"] = entry.Value
				item["resourceType"] = entry.ResourceType
				item["resourceMountPoint"] = entry.ResourceMountPoint
				item["resourceKey"] = entry.ResourceKey
				result = append(result, item)
			}
		}
	}

	return result
}

func collectConfigurationPairs(configurationType string, configurable ...v1.Configurable) []variable {
	result := make([]variable, 0)

	for _, c := range configurable {
		c := c

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

func mustHomeDir() string {
	dir, err := user.Dir()
	if err != nil {
		panic(err)
	}
	return dir
}

func toHostDir(host string) string {
	h := strings.Replace(strings.Replace(host, "https://", "", 1), "http://", "", 1)
	return toFileName.ReplaceAllString(h, "_")
}

func AddSourceDependencies(source v1.SourceSpec, catalog *camel.RuntimeCatalog) *strset.Set {
	dependencies := strset.New()

	// Add auto-detected dependencies
	meta := metadata.Extract(catalog, source)
	dependencies.Merge(meta.Dependencies)

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
func AssertTraitsType(traits interface{}) error {
	_, ok1 := traits.(v1.Traits)
	_, ok2 := traits.(v1.IntegrationKitTraits)
	if !ok1 && !ok2 {
		return errors.New("traits must be either v1.Traits or v1.IntegrationKitTraits")
	}

	return nil
}

// ToTraitMap accepts either v1.Traits or v1.IntegrationKitTraits and converts it to a map of traits.
func ToTraitMap(traits interface{}) (Options, error) {
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

// ToPropertyMap accepts a trait and converts it to a map of trait properties.
func ToPropertyMap(trait interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(trait)
	if err != nil {
		return nil, err
	}
	propMap := make(map[string]interface{})
	if err = json.Unmarshal(data, &propMap); err != nil {
		return nil, err
	}

	return propMap, nil
}

// MigrateLegacyConfiguration moves up the legacy configuration in a trait to the new top-level properties.
// Values of the new properties always take precedence over the ones from the legacy configuration
// with the same property names.
func MigrateLegacyConfiguration(trait map[string]interface{}) error {
	if trait["configuration"] == nil {
		return nil
	}

	if config, ok := trait["configuration"].(map[string]interface{}); ok {
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
		return errors.Errorf(`unexpected type for "configuration" field: %v`, reflect.TypeOf(trait["configuration"]))
	}

	return nil
}

// ToTrait unmarshals a map configuration to a target trait.
func ToTrait(trait map[string]interface{}, target interface{}) error {
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

// Equals return if traits are the same.
func Equals(i1 Options, i2 Options) bool {
	return reflect.DeepEqual(i1, i2)
}

// IntegrationsHaveSameTraits return if traits are the same.
func IntegrationsHaveSameTraits(i1 *v1.Integration, i2 *v1.Integration) (bool, error) {
	c1, err := NewTraitsOptionsForIntegration(i1)
	if err != nil {
		return false, err
	}
	c2, err := NewTraitsOptionsForIntegration(i2)
	if err != nil {
		return false, err
	}

	return Equals(c1, c2), nil
}

// IntegrationKitsHaveSameTraits return if traits are the same.
func IntegrationKitsHaveSameTraits(i1 *v1.IntegrationKit, i2 *v1.IntegrationKit) (bool, error) {
	c1, err := NewTraitsOptionsForIntegrationKit(i1)
	if err != nil {
		return false, err
	}
	c2, err := NewTraitsOptionsForIntegrationKit(i2)
	if err != nil {
		return false, err
	}

	return Equals(c1, c2), nil
}

// KameletBindingsHaveSameTraits return if traits are the same.
func KameletBindingsHaveSameTraits(i1 *v1alpha1.KameletBinding, i2 *v1alpha1.KameletBinding) (bool, error) {
	c1, err := NewTraitsOptionsForKameletBinding(i1)
	if err != nil {
		return false, err
	}
	c2, err := NewTraitsOptionsForKameletBinding(i2)
	if err != nil {
		return false, err
	}

	return Equals(c1, c2), nil
}

// IntegrationAndBindingSameTraits return if traits are the same.
// The comparison is done for the subset of traits defines on the binding as during the trait processing,
// some traits may be added to the Integration i.e. knative configuration in case of sink binding.
func IntegrationAndBindingSameTraits(i1 *v1.Integration, i2 *v1alpha1.KameletBinding) (bool, error) {
	itOpts, err := NewTraitsOptionsForIntegration(i1)
	if err != nil {
		return false, err
	}
	klbOpts, err := NewTraitsOptionsForKameletBinding(i2)
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

// IntegrationAndKitHaveSameTraits return if traits are the same.
func IntegrationAndKitHaveSameTraits(i1 *v1.Integration, i2 *v1.IntegrationKit) (bool, error) {
	itOpts, err := NewTraitsOptionsForIntegration(i1)
	if err != nil {
		return false, err
	}
	ikOpts, err := NewTraitsOptionsForIntegrationKit(i2)
	if err != nil {
		return false, err
	}

	return Equals(ikOpts, itOpts), nil
}

func NewTraitsOptionsForIntegration(i *v1.Integration) (Options, error) {
	m1, err := ToTraitMap(i.Spec.Traits)
	if err != nil {
		return nil, err
	}

	m2, err := FromAnnotations(&i.ObjectMeta)
	if err != nil {
		return nil, err
	}

	for k, v := range m2 {
		m1[k] = v
	}

	return m1, nil
}

func NewTraitsOptionsForIntegrationKit(i *v1.IntegrationKit) (Options, error) {
	m1, err := ToTraitMap(i.Spec.Traits)
	if err != nil {
		return nil, err
	}

	m2, err := FromAnnotations(&i.ObjectMeta)
	if err != nil {
		return nil, err
	}

	for k, v := range m2 {
		m1[k] = v
	}

	return m1, nil
}

func NewTraitsOptionsForIntegrationPlatform(i *v1.IntegrationPlatform) (Options, error) {
	m1, err := ToTraitMap(i.Spec.Traits)
	if err != nil {
		return nil, err
	}

	m2, err := FromAnnotations(&i.ObjectMeta)
	if err != nil {
		return nil, err
	}

	for k, v := range m2 {
		m1[k] = v
	}

	return m1, nil
}

func NewTraitsOptionsForKameletBinding(i *v1alpha1.KameletBinding) (Options, error) {
	if i.Spec.Integration != nil {
		m1, err := ToTraitMap(i.Spec.Integration.Traits)
		if err != nil {
			return nil, err
		}

		m2, err := FromAnnotations(&i.ObjectMeta)
		if err != nil {
			return nil, err
		}

		for k, v := range m2 {
			m1[k] = v
		}

		return m1, nil
	}

	m1, err := FromAnnotations(&i.ObjectMeta)
	if err != nil {
		return nil, err
	}

	return m1, nil
}

func FromAnnotations(meta *metav1.ObjectMeta) (Options, error) {
	options := make(Options)

	for k, v := range meta.Annotations {
		if strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
			configKey := strings.TrimPrefix(k, v1.TraitAnnotationPrefix)
			if strings.Contains(configKey, ".") {
				parts := strings.SplitN(configKey, ".", 2)
				id := parts[0]
				prop := parts[1]
				if _, ok := options[id]; !ok {
					options[id] = make(map[string]interface{})
				}

				propParts := util.ConfigTreePropertySplit(prop)
				var current = options[id]
				if len(propParts) > 1 {
					c, err := util.NavigateConfigTree(current, propParts[0:len(propParts)-1])
					if err != nil {
						return options, err
					}
					if cc, ok := c.(map[string]interface{}); ok {
						current = cc
					} else {
						return options, errors.New(`invalid array specification: to set an array value use the ["v1", "v2"] format`)
					}
				}
				current[prop] = v

			} else {
				return options, fmt.Errorf("wrong format for trait annotation %q: missing trait ID", k)
			}
		}
	}

	return options, nil
}
