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

package keda

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"

	kedav1alpha1 "github.com/apache/camel-k/addons/keda/duck/v1alpha1"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	camelv1alpha1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/kamelet/repository"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/property"
	"github.com/apache/camel-k/pkg/util/source"
	"github.com/apache/camel-k/pkg/util/uri"
	"github.com/pkg/errors"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// kameletURNMetadataPrefix allows binding Kamelet properties to KEDA metadata.
	kameletURNMetadataPrefix = "urn:keda:metadata:"
	// kameletURNAuthenticationPrefix allows binding Kamelet properties to KEDA authentication options.
	kameletURNAuthenticationPrefix = "urn:keda:authentication:"
	// kameletURNRequiredTag is used to mark properties required by KEDA.
	kameletURNRequiredTag = "urn:keda:required"

	// kameletAnnotationType indicates the scaler type associated to a Kamelet.
	kameletAnnotationType = "camel.apache.org/keda.type"
	// kameletAnnotationMetadataPrefix is used to define virtual metadata fields computed from Kamelet properties.
	kameletAnnotationMetadataPrefix = "camel.apache.org/keda.metadata."
	// kameletAnnotationAuthenticationPrefix is used to define virtual authentication fields computed from Kamelet properties.
	kameletAnnotationAuthenticationPrefix = "camel.apache.org/keda.authentication."
)

// The KEDA trait can be used for automatic integration with KEDA autoscalers.
// The trait can be either manually configured using the `triggers` option or automatically configured
// via markers in the Kamelets.
//
// For information on how to use KEDA enabled Kamelets with the KEDA trait, refer to
// xref:ROOT:kamelets/kamelets-user.adoc#kamelet-keda-user[the KEDA section in the Kamelets user guide].
// If you want to create Kamelets that contain KEDA metadata, refer to
// xref:ROOT:kamelets/kamelets-dev.adoc#kamelet-keda-dev[the KEDA section in the Kamelets development guide].
//
// The KEDA trait is disabled by default.
//
// +camel-k:trait=keda.
type kedaTrait struct {
	trait.BaseTrait `property:",squash"`
	// Enables automatic configuration of the trait. Allows the trait to infer KEDA triggers from the Kamelets.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Set the spec->replicas field on the top level controller to an explicit value if missing, to allow KEDA to recognize it as a scalable resource.
	HackControllerReplicas *bool `property:"hack-controller-replicas" json:"hackControllerReplicas,omitempty"`
	// Interval (seconds) to check each trigger on.
	PollingInterval *int32 `property:"polling-interval" json:"pollingInterval,omitempty"`
	// The wait period between the last active trigger reported and scaling the resource back to 0.
	CooldownPeriod *int32 `property:"cooldown-period" json:"cooldownPeriod,omitempty"`
	// Enabling this property allows KEDA to scale the resource down to the specified number of replicas.
	IdleReplicaCount *int32 `property:"idle-replica-count" json:"idleReplicaCount,omitempty"`
	// Minimum number of replicas.
	MinReplicaCount *int32 `property:"min-replica-count" json:"minReplicaCount,omitempty"`
	// Maximum number of replicas.
	MaxReplicaCount *int32 `property:"max-replica-count" json:"maxReplicaCount,omitempty"`
	// Definition of triggers according to the KEDA format. Each trigger must contain `type` field corresponding
	// to the name of a KEDA autoscaler and a key/value map named `metadata` containing specific trigger options.
	// An optional `authentication-secret` can be declared per trigger and the operator will link each entry of
	// the secret to a KEDA authentication parameter.
	Triggers []kedaTrigger `property:"triggers" json:"triggers,omitempty"`
}

type kedaTrigger struct {
	Type                 string            `property:"type" json:"type,omitempty"`
	Metadata             map[string]string `property:"metadata" json:"metadata,omitempty"`
	AuthenticationSecret string            `property:"authentication-secret" json:"authenticationSecret,omitempty"`
	authentication       map[string]string
}

// NewKedaTrait --.
func NewKedaTrait() trait.Trait {
	return &kedaTrait{
		BaseTrait: trait.NewBaseTrait("keda", trait.TraitOrderPostProcessResources),
	}
}

func (t *kedaTrait) Configure(e *trait.Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(camelv1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if err := t.populateTriggersFromKamelets(e); err != nil {
			return false, err
		}
	}

	return len(t.Triggers) > 0, nil
}

func (t *kedaTrait) Apply(e *trait.Environment) error {
	if e.IntegrationInPhase(camelv1.IntegrationPhaseInitialization) {
		if t.HackControllerReplicas == nil || *t.HackControllerReplicas {
			if err := t.hackControllerReplicas(e); err != nil {
				return err
			}
		}
	} else if e.IntegrationInRunningPhases() {
		if err := t.addScalingResources(e); err != nil {
			return err
		}
	}

	return nil
}

func (t *kedaTrait) addScalingResources(e *trait.Environment) error {
	if len(t.Triggers) == 0 {
		return nil
	}

	obj := kedav1alpha1.NewScaledObject(e.Integration.Namespace, e.Integration.Name)
	obj.Spec.ScaleTargetRef = t.getTopControllerReference(e)
	if t.PollingInterval != nil {
		obj.Spec.PollingInterval = t.PollingInterval
	}
	if t.CooldownPeriod != nil {
		obj.Spec.CooldownPeriod = t.CooldownPeriod
	}
	if t.IdleReplicaCount != nil {
		obj.Spec.IdleReplicaCount = t.IdleReplicaCount
	}
	if t.MinReplicaCount != nil {
		obj.Spec.MinReplicaCount = t.MinReplicaCount
	}
	if t.MaxReplicaCount != nil {
		obj.Spec.MaxReplicaCount = t.MaxReplicaCount
	}
	for idx, trigger := range t.Triggers {
		meta := make(map[string]string)
		for k, v := range trigger.Metadata {
			meta[k] = v
		}
		var authenticationRef *kedav1alpha1.ScaledObjectAuthRef
		if len(trigger.authentication) > 0 && trigger.AuthenticationSecret != "" {
			return errors.New("an authentication secret cannot be provided for auto-configured triggers")
		}
		extConfigName := fmt.Sprintf("%s-keda-%d", e.Integration.Name, idx)
		if len(trigger.authentication) > 0 {
			// Save all authentication config in a secret
			secret := v1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: v1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: e.Integration.Namespace,
					Name:      extConfigName,
				},
				StringData: trigger.authentication,
			}
			e.Resources.Add(&secret)

			// Link the secret using a TriggerAuthentication
			triggerAuth := kedav1alpha1.TriggerAuthentication{
				TypeMeta: metav1.TypeMeta{
					Kind:       "TriggerAuthentication",
					APIVersion: kedav1alpha1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: e.Integration.Namespace,
					Name:      extConfigName,
				},
			}
			for _, k := range util.SortedStringMapKeys(trigger.authentication) {
				triggerAuth.Spec.SecretTargetRef = append(triggerAuth.Spec.SecretTargetRef, kedav1alpha1.AuthSecretTargetRef{
					Parameter: k,
					Name:      extConfigName,
					Key:       k,
				})
			}
			e.Resources.Add(&triggerAuth)
			authenticationRef = &kedav1alpha1.ScaledObjectAuthRef{
				Name: extConfigName,
			}
		} else if trigger.AuthenticationSecret != "" {
			s := v1.Secret{}
			key := ctrl.ObjectKey{
				Namespace: e.Integration.Namespace,
				Name:      trigger.AuthenticationSecret,
			}
			if err := e.Client.Get(e.Ctx, key, &s); err != nil {
				return errors.Wrapf(err, "could not load secret named %q in namespace %q", trigger.AuthenticationSecret, e.Integration.Namespace)
			}
			// Fill a TriggerAuthentication from the secret
			triggerAuth := kedav1alpha1.TriggerAuthentication{
				TypeMeta: metav1.TypeMeta{
					Kind:       "TriggerAuthentication",
					APIVersion: kedav1alpha1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: e.Integration.Namespace,
					Name:      extConfigName,
				},
			}
			sortedKeys := make([]string, 0, len(s.Data))
			for k := range s.Data {
				sortedKeys = append(sortedKeys, k)
			}
			sort.Strings(sortedKeys)
			for _, k := range sortedKeys {
				triggerAuth.Spec.SecretTargetRef = append(triggerAuth.Spec.SecretTargetRef, kedav1alpha1.AuthSecretTargetRef{
					Parameter: k,
					Name:      s.Name,
					Key:       k,
				})
			}
			e.Resources.Add(&triggerAuth)
			authenticationRef = &kedav1alpha1.ScaledObjectAuthRef{
				Name: extConfigName,
			}
		}

		st := kedav1alpha1.ScaleTriggers{
			Type:              trigger.Type,
			Metadata:          meta,
			AuthenticationRef: authenticationRef,
		}
		obj.Spec.Triggers = append(obj.Spec.Triggers, st)
	}
	e.Resources.Add(&obj)
	return nil
}

func (t *kedaTrait) hackControllerReplicas(e *trait.Environment) error {
	ctrlRef := t.getTopControllerReference(e)
	scale := autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: int32(1),
		},
	}
	scalesClient, err := e.Client.ScalesClient()
	if err != nil {
		return err
	}
	if ctrlRef.Kind == camelv1alpha1.KameletBindingKind {
		scale.ObjectMeta.Name = ctrlRef.Name
		_, err = scalesClient.Scales(e.Integration.Namespace).Update(e.Ctx, camelv1alpha1.SchemeGroupVersion.WithResource("kameletbindings").GroupResource(), &scale, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	} else if e.Integration.Spec.Replicas == nil {
		scale.ObjectMeta.Name = e.Integration.Name
		_, err = scalesClient.Scales(e.Integration.Namespace).Update(e.Ctx, camelv1.SchemeGroupVersion.WithResource("integrations").GroupResource(), &scale, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *kedaTrait) getTopControllerReference(e *trait.Environment) *v1.ObjectReference {
	for _, o := range e.Integration.OwnerReferences {
		if o.Kind == camelv1alpha1.KameletBindingKind && strings.HasPrefix(o.APIVersion, camelv1alpha1.SchemeGroupVersion.Group) {
			return &v1.ObjectReference{
				APIVersion: o.APIVersion,
				Kind:       o.Kind,
				Name:       o.Name,
			}
		}
	}
	return &v1.ObjectReference{
		APIVersion: e.Integration.APIVersion,
		Kind:       e.Integration.Kind,
		Name:       e.Integration.Name,
	}
}

func (t *kedaTrait) populateTriggersFromKamelets(e *trait.Environment) error {
	sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
	if err != nil {
		return err
	}
	kameletURIs := make(map[string][]string)
	metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
		for _, kameletURI := range meta.FromURIs {
			if kameletStr := source.ExtractKamelet(kameletURI); kameletStr != "" && camelv1alpha1.ValidKameletName(kameletStr) {
				kamelet := kameletStr
				if strings.Contains(kamelet, "/") {
					kamelet = kamelet[0:strings.Index(kamelet, "/")]
				}
				uriList := kameletURIs[kamelet]
				util.StringSliceUniqueAdd(&uriList, kameletURI)
				sort.Strings(uriList)
				kameletURIs[kamelet] = uriList
			}
		}
		return true
	})

	if len(kameletURIs) == 0 {
		return nil
	}

	repo, err := repository.NewForPlatform(e.Ctx, e.Client, e.Platform, e.Integration.Namespace, platform.GetOperatorNamespace())
	if err != nil {
		return err
	}

	sortedKamelets := make([]string, 0, len(kameletURIs))
	for kamelet := range kameletURIs {
		sortedKamelets = append(sortedKamelets, kamelet)
	}
	sort.Strings(sortedKamelets)
	for _, kamelet := range sortedKamelets {
		uris := kameletURIs[kamelet]
		if err := t.populateTriggersFromKamelet(e, repo, kamelet, uris); err != nil {
			return err
		}
	}

	return nil
}

func (t *kedaTrait) populateTriggersFromKamelet(e *trait.Environment, repo repository.KameletRepository, kameletName string, uris []string) error {
	kamelet, err := repo.Get(e.Ctx, kameletName)
	if err != nil {
		return err
	} else if kamelet == nil {
		return fmt.Errorf("kamelet %q not found", kameletName)
	}
	if kamelet.Spec.Definition == nil {
		return nil
	}
	triggerType := kamelet.Annotations[kameletAnnotationType]
	if triggerType == "" {
		return nil
	}

	kedaParamToProperty := make(map[string]string)
	requiredKEDAParam := make(map[string]bool)
	kedaAuthenticationParam := make(map[string]bool)
	for k, def := range kamelet.Spec.Definition.Properties {
		if metadataName := t.getXDescriptorValue(def.XDescriptors, kameletURNMetadataPrefix); metadataName != "" {
			kedaParamToProperty[metadataName] = k
			if req := t.isXDescriptorPresent(def.XDescriptors, kameletURNRequiredTag); req {
				requiredKEDAParam[metadataName] = true
			}
		}
		if authenticationName := t.getXDescriptorValue(def.XDescriptors, kameletURNAuthenticationPrefix); authenticationName != "" {
			kedaParamToProperty[authenticationName] = k
			if req := t.isXDescriptorPresent(def.XDescriptors, kameletURNRequiredTag); req {
				requiredKEDAParam[authenticationName] = true
			}
			kedaAuthenticationParam[authenticationName] = true
		}
	}
	for _, kameletURI := range uris {
		if err := t.populateTriggersFromKameletURI(e, kamelet, triggerType, kedaParamToProperty, requiredKEDAParam, kedaAuthenticationParam, kameletURI); err != nil {
			return err
		}
	}
	return nil
}

func (t *kedaTrait) populateTriggersFromKameletURI(e *trait.Environment, kamelet *camelv1alpha1.Kamelet, triggerType string, kedaParamToProperty map[string]string, requiredKEDAParam map[string]bool, authenticationParams map[string]bool, kameletURI string) error {
	metaValues := make(map[string]string, len(kedaParamToProperty))
	for metaParam, prop := range kedaParamToProperty {
		v, err := t.getKameletPropertyValue(e, kamelet, kameletURI, prop)
		if err != nil {
			return err
		}
		if v != "" {
			metaValues[metaParam] = v
		}
	}

	metaTemplates, templateAuthParams, err := t.evaluateTemplateParameters(e, kamelet, kameletURI)
	if err != nil {
		return err
	}
	for k, v := range metaTemplates {
		metaValues[k] = v
	}
	for k, v := range templateAuthParams {
		authenticationParams[k] = v
	}

	for req := range requiredKEDAParam {
		if _, ok := metaValues[req]; !ok {
			return fmt.Errorf("metadata parameter %q is missing in configuration: it is required by KEDA", req)
		}
	}

	onlyMetaValues := make(map[string]string, len(metaValues)-len(authenticationParams))
	onlyAuthValues := make(map[string]string, len(authenticationParams))
	for k, v := range metaValues {
		if authenticationParams[k] {
			onlyAuthValues[k] = v
		} else {
			onlyMetaValues[k] = v
		}
	}

	// Add the trigger in config
	trigger := kedaTrigger{
		Type:           triggerType,
		Metadata:       onlyMetaValues,
		authentication: onlyAuthValues,
	}
	t.Triggers = append(t.Triggers, trigger)
	return nil
}

func (t *kedaTrait) evaluateTemplateParameters(e *trait.Environment, kamelet *camelv1alpha1.Kamelet, kameletURI string) (map[string]string, map[string]bool, error) {
	paramTemplates := make(map[string]string)
	authenticationParam := make(map[string]bool)
	for annotation, expr := range kamelet.Annotations {
		if strings.HasPrefix(annotation, kameletAnnotationMetadataPrefix) {
			paramName := annotation[len(kameletAnnotationMetadataPrefix):]
			paramTemplates[paramName] = expr
		} else if strings.HasPrefix(annotation, kameletAnnotationAuthenticationPrefix) {
			paramName := annotation[len(kameletAnnotationAuthenticationPrefix):]
			paramTemplates[paramName] = expr
			authenticationParam[paramName] = true
		}
	}

	kameletPropValues := make(map[string]string)
	if kamelet.Spec.Definition != nil {
		for prop := range kamelet.Spec.Definition.Properties {
			val, err := t.getKameletPropertyValue(e, kamelet, kameletURI, prop)
			if err != nil {
				return nil, nil, err
			}
			if val != "" {
				kameletPropValues[prop] = val
			}
		}
	}

	paramValues := make(map[string]string, len(paramTemplates))
	for param, expr := range paramTemplates {
		tmpl, err := template.New(fmt.Sprintf("kamelet-param-%s", param)).Parse(expr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "invalid template for KEDA parameter %q: %q", param, expr)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, kameletPropValues); err != nil {
			return nil, nil, errors.Wrapf(err, "unable to process template for KEDA parameter %q: %q", param, expr)
		}
		paramValues[param] = buf.String()
	}
	return paramValues, authenticationParam, nil
}

func (t *kedaTrait) getKameletPropertyValue(e *trait.Environment, kamelet *camelv1alpha1.Kamelet, kameletURI, prop string) (string, error) {
	// From top priority to lowest
	if v := uri.GetQueryParameter(kameletURI, prop); v != "" {
		return v, nil
	}
	if kameletID := uri.GetPathSegment(kameletURI, 0); kameletID != "" {
		kameletSpecificKey := fmt.Sprintf("camel.kamelet.%s.%s.%s", kamelet.Name, kameletID, prop)
		for _, c := range e.Integration.Spec.Configuration {
			if c.Type == "property" && strings.HasPrefix(c.Value, kameletSpecificKey) {
				v, err := property.DecodePropertyFileValue(c.Value, kameletSpecificKey)
				if err != nil {
					return "", errors.Wrapf(err, "could not decode property %q", kameletSpecificKey)
				}
				return v, nil
			}
		}

		if v := e.ApplicationProperties[kameletSpecificKey]; v != "" {
			return v, nil
		}

	}
	if v := e.ApplicationProperties[fmt.Sprintf("camel.kamelet.%s.%s", kamelet.Name, prop)]; v != "" {
		return v, nil
	}
	if kamelet.Spec.Definition != nil {
		if schema, ok := kamelet.Spec.Definition.Properties[prop]; ok && schema.Default != nil {
			var val interface{}
			d := json.NewDecoder(bytes.NewReader(schema.Default.RawMessage))
			d.UseNumber()
			if err := d.Decode(&val); err != nil {
				return "", errors.Wrapf(err, "cannot decode default value for property %q", prop)
			}
			v := fmt.Sprintf("%v", val)
			return v, nil
		}
	}
	return "", nil
}

func (t *kedaTrait) getXDescriptorValue(descriptors []string, prefix string) string {
	for _, d := range descriptors {
		if strings.HasPrefix(d, prefix) {
			return d[len(prefix):]
		}
	}
	return ""
}

func (t *kedaTrait) isXDescriptorPresent(descriptors []string, desc string) bool {
	for _, d := range descriptors {
		if d == desc {
			return true
		}
	}
	return false
}
