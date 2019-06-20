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
	"strconv"
	"strings"

	"github.com/scylladb/go-set/strset"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/log"
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
}

/* Base trait */

func newBaseTrait(id string) BaseTrait {
	return BaseTrait{
		id: ID(id),
		L:  log.Log.WithName("traits").WithValues("trait", id),
	}
}

// BaseTrait is the root trait with noop implementations for hooks
type BaseTrait struct {
	id      ID
	Enabled *bool `property:"enabled"`
	client  client.Client
	ctx     context.Context
	L       log.Logger
}

// ID returns the identifier of the trait
func (trait *BaseTrait) ID() ID {
	return trait.id
}

// InjectClient implements client.ClientInject and allows to inject a client into the trait
func (trait *BaseTrait) InjectClient(c client.Client) {
	trait.client = c
}

// InjectContext allows to inject a context into the trait
func (trait *BaseTrait) InjectContext(ctx context.Context) {
	trait.ctx = ctx
}

/* Environment */

// A Environment provides the context where the trait is executed
type Environment struct {
	CamelCatalog   *camel.RuntimeCatalog
	RuntimeVersion string
	Catalog        *Catalog
	C              context.Context
	Client         client.Client
	Platform       *v1alpha1.IntegrationPlatform
	IntegrationKit *v1alpha1.IntegrationKit
	Integration    *v1alpha1.Integration
	Resources      *kubernetes.Collection
	PostActions    []func(*Environment) error
	PostProcessors []func(*Environment) error
	Steps          []builder.Step
	BuildDir       string
	ExecutedTraits []Trait
	EnvVars        []corev1.EnvVar
	Classpath      *strset.Set
}

// ControllerStrategy is used to determine the kind of controller that needs to be created for the integration
type ControllerStrategy string

// List of controller strategies
const (
	ControllerStrategyDeployment     = "deployment"
	ControllerStrategyKnativeService = "knative-service"
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
func (e *Environment) IntegrationInPhase(phases ...v1alpha1.IntegrationPhase) bool {
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
func (e *Environment) IntegrationKitInPhase(phases ...v1alpha1.IntegrationKitPhase) bool {
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
func (e *Environment) InPhase(c v1alpha1.IntegrationKitPhase, i v1alpha1.IntegrationPhase) bool {
	return e.IntegrationKitInPhase(c) && e.IntegrationInPhase(i)
}

// DetermineProfile determines the TraitProfile of the environment.
// First looking at the Integration.Spec for a Profile,
// next looking at the IntegrationKit.Spec
// and lastly the Platform Profile
func (e *Environment) DetermineProfile() v1alpha1.TraitProfile {
	if e.Integration != nil && e.Integration.Spec.Profile != "" {
		return e.Integration.Spec.Profile
	}

	if e.IntegrationKit != nil && e.IntegrationKit.Spec.Profile != "" {
		return e.IntegrationKit.Spec.Profile
	}

	return platform.GetProfile(e.Platform)
}

// DetermineControllerStrategy determines the type of controller that should be used for the integration
func (e *Environment) DetermineControllerStrategy(ctx context.Context, c client.Client) (ControllerStrategy, error) {
	if e.DetermineProfile() != v1alpha1.TraitProfileKnative {
		return ControllerStrategyDeployment, nil
	}

	trait := e.GetTrait("deployer")
	if trait != nil {
		deployerTrait := trait.(*deployerTrait)
		if deployerTrait.Kind == ControllerStrategyDeployment {
			return ControllerStrategyDeployment, nil
		} else if deployerTrait.Kind == ControllerStrategyKnativeService {
			return ControllerStrategyKnativeService, nil
		}
	}

	var sources []v1alpha1.SourceSpec
	var err error
	if sources, err = kubernetes.ResolveIntegrationSources(ctx, c, e.Integration, e.Resources); err != nil {
		return "", err
	}

	// In Knative profile: use knative service only if needed
	meta := metadata.ExtractAll(e.CamelCatalog, sources)
	if !meta.RequiresHTTPService {
		return ControllerStrategyDeployment, nil
	}

	return ControllerStrategyKnativeService, nil
}

// DetermineCamelVersion --
func (e *Environment) DetermineCamelVersion() string {
	var version string

	if e.Integration != nil {
		version = e.Integration.Status.CamelVersion
	}
	if e.IntegrationKit != nil && version == "" {
		version = e.IntegrationKit.Status.CamelVersion
	}
	if version == "" {
		version = e.Platform.Spec.Build.CamelVersion
	}

	return version
}

// DetermineRuntimeVersion --
func (e *Environment) DetermineRuntimeVersion() string {
	var version string

	if e.Integration != nil {
		version = e.Integration.Status.RuntimeVersion
	}
	if e.IntegrationKit != nil && version == "" {
		version = e.IntegrationKit.Status.RuntimeVersion
	}
	if version == "" {
		version = e.Platform.Spec.Build.RuntimeVersion
	}

	return version
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
		if r.Type != v1alpha1.ResourceTypeData {
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
		root := "/etc/camel/sources"
		root = path.Join(root, fmt.Sprintf("i-source-%03d", i))

		srcName := strings.TrimPrefix(s.Name, "/")
		src := path.Join(root, srcName)
		src = "file:" + src

		params := make([]string, 0)
		if s.InferLanguage() != "" {
			params = append(params, "language="+string(s.InferLanguage()))
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
		resPath := path.Join("/etc/camel/sources", refName)

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

	for i, r := range e.Integration.Spec.Resources {
		if r.Type != v1alpha1.ResourceTypeData {
			continue
		}

		cmName := fmt.Sprintf("%s-resource-%03d", e.Integration.Name, i)
		refName := fmt.Sprintf("i-resource-%03d", i)
		resName := strings.TrimPrefix(r.Name, "/")
		cmKey := "content"
		resPath := path.Join("/etc/camel/resources", refName)

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
		MountPath: "/etc/camel/conf",
	})

	//
	// Volumes :: Additional ConfigMaps
	//

	for _, cmName := range e.CollectConfigurationValues("configmap") {
		refName := kubernetes.SanitizeLabel(cmName)
		fileName := "integration-cm-" + strings.ToLower(cmName)

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
			MountPath: path.Join("/etc/camel/conf.d", fileName),
		})
	}

	//
	// Volumes :: Additional Secrets
	//

	for _, secretName := range e.CollectConfigurationValues("secret") {
		refName := kubernetes.SanitizeLabel(secretName)
		fileName := "integration-secret-" + strings.ToLower(secretName)

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
			MountPath: path.Join("/etc/camel/conf.d", fileName),
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
