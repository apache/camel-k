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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/io"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/gzip"
	"github.com/apache/camel-k/v2/pkg/util/jvm"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/maven"
)

const (
	openapiTraitID    = "openapi"
	openapiTraitOrder = 300
)

type openAPITrait struct {
	BasePlatformTrait
	traitv1.OpenAPITrait `property:",squash"`
}

func newOpenAPITrait() Trait {
	return &openAPITrait{
		BasePlatformTrait: NewBasePlatformTrait(openapiTraitID, openapiTraitOrder),
	}
}

func (t *openAPITrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if t.Configmaps != nil {
		if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
			condition := NewIntegrationCondition(
				"OpenApi",
				v1.IntegrationConditionTraitInfo,
				corev1.ConditionTrue,
				TraitConfigurationReason,
				"OpenApi trait is deprecated and may be removed in future version: "+
					"use Camel REST contract first instead, https://camel.apache.org/manual/rest-dsl-openapi.html",
			)

			return true, condition, nil
		}
	}

	return false, nil, nil
}

func (t *openAPITrait) Apply(e *Environment) error {
	util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityRest)

	root := os.TempDir()
	tmpDir, err := os.MkdirTemp(root, "openapi")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	generatedFromConfigmaps, err := t.generateFromConfigmaps(e, tmpDir)
	if err != nil {
		return err
	}
	e.Integration.Status.GeneratedSources = generatedFromConfigmaps

	return nil
}

func (t *openAPITrait) generateFromConfigmaps(e *Environment, tmpDir string) ([]v1.SourceSpec, error) {
	dataSpecs := make([]v1.DataSpec, 0, len(t.Configmaps))
	for _, configmap := range t.Configmaps {
		cm := kubernetes.LookupConfigmap(e.Ctx, e.Client, e.Integration.Namespace, configmap)
		if cm == nil {
			return nil, fmt.Errorf("configmap %s does not exist in namespace %s", configmap, e.Integration.Namespace)
		}
		// Iterate over each configmap key which may hold a different OpenAPI spec
		for k, v := range cm.Data {
			dataSpecs = append(dataSpecs, v1.DataSpec{
				Name:        k,
				Content:     v,
				Compression: false,
			})

		}
	}

	return t.generateFromDataSpecs(e, tmpDir, dataSpecs)
}

func (t *openAPITrait) generateFromDataSpecs(e *Environment, tmpDir string, specs []v1.DataSpec) ([]v1.SourceSpec, error) {
	generatedSources := make([]v1.SourceSpec, 0, len(e.Integration.Status.GeneratedSources))
	for i, resource := range specs {
		generatedContentName := fmt.Sprintf("%s-openapi-%03d", e.Integration.Name, i)
		generatedSourceName := strings.TrimSuffix(resource.Name, filepath.Ext(resource.Name)) + ".xml"
		// Generate configmap or reuse existing one
		if err := t.generateOpenAPIConfigMap(e, resource, tmpDir, generatedContentName); err != nil {
			return nil, fmt.Errorf("cannot generate configmap for openapi resource %s: %w", resource.Name, err)
		}
		if e.Integration.Status.GeneratedSources != nil {
			// Filter out the previously generated source
			for _, x := range e.Integration.Status.GeneratedSources {
				if x.Name != generatedSourceName {
					generatedSources = append(generatedSources, x)
				}
			}
		}

		// Add a source that references the config map
		generatedSources = append(generatedSources, v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:        generatedSourceName,
				ContentRef:  generatedContentName,
				Compression: resource.Compression,
			},
			Language: v1.LanguageXML,
		})
	}

	return generatedSources, nil
}

func (t *openAPITrait) generateOpenAPIConfigMap(e *Environment, resource v1.DataSpec, tmpDir, generatedContentName string) error {
	cm := corev1.ConfigMap{}
	key := ctrl.ObjectKey{
		Namespace: e.Integration.Namespace,
		Name:      generatedContentName,
	}
	err := t.Client.Get(e.Ctx, key, &cm)
	if err != nil && k8serrors.IsNotFound(err) {
		return t.createNewOpenAPIConfigMap(e, resource, tmpDir, generatedContentName)
	} else if err != nil {
		return err
	}

	// ConfigMap already present, let's check if the source has not changed
	foundDigest := cm.Annotations["camel.apache.org/source.digest"]

	// Compute the new digest
	newDigest, err := digest.ComputeForResource(resource)
	if err != nil {
		return err
	}

	if foundDigest == newDigest {
		// ConfigMap already exists and matches the source
		// Re-adding it to update its revision
		cm.ResourceVersion = ""
		// Clear the managed fields to support server-side apply
		cm.ManagedFields = nil
		e.Resources.Add(&cm)
		return nil
	}
	return t.createNewOpenAPIConfigMap(e, resource, tmpDir, generatedContentName)
}

func (t *openAPITrait) createNewOpenAPIConfigMap(e *Environment, resource v1.DataSpec, tmpDir, generatedContentName string) error {
	tmpDir = filepath.Join(tmpDir, generatedContentName)
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		return err
	}

	content := []byte(resource.Content)
	if resource.Compression {
		content, err = gzip.UncompressBase64(content)
		if err != nil {
			return err
		}
	}

	in := filepath.Join(tmpDir, resource.Name)
	out := filepath.Join(tmpDir, "openapi-dsl.xml")

	err = os.WriteFile(in, content, io.FilePerm400)
	if err != nil {
		return err
	}

	project := t.generateMavenProject(e.CamelCatalog.Runtime.Version)
	mc := maven.NewContext(tmpDir)
	mc.LocalRepository = e.Platform.Status.Build.Maven.LocalRepository
	mc.AdditionalArguments = e.Platform.Status.Build.Maven.CLIOptions
	mc.AddArgument("-Dopenapi.spec=" + in)
	mc.AddArgument("-Ddsl.out=" + out)

	if settings, err := kubernetes.ResolveValueSource(e.Ctx, e.Client, e.Platform.Namespace, &e.Platform.Status.Build.Maven.Settings); err != nil {
		return err
	} else if settings != "" {
		mc.UserSettings = []byte(settings)
	}

	settings, err := maven.NewSettings(maven.DefaultRepositories, maven.ProxyFromEnvironment)
	if err != nil {
		return err
	}
	data, err := settings.MarshalBytes()
	if err != nil {
		return err
	}
	mc.GlobalSettings = data
	secrets := e.Platform.Status.Build.Maven.CASecrets

	if secrets != nil {
		certsData, err := kubernetes.GetSecretsRefData(e.Ctx, e.Client, e.Platform.Namespace, secrets)
		if err != nil {
			return err
		}
		trustStoreName := "trust.jks"
		trustStorePass := jvm.NewKeystorePassword()
		err = jvm.GenerateKeystore(e.Ctx, tmpDir, trustStoreName, trustStorePass, certsData)
		if err != nil {
			return err
		}
		mc.ExtraMavenOpts = append(mc.ExtraMavenOpts,
			"-Djavax.net.ssl.trustStore="+trustStoreName,
			"-Djavax.net.ssl.trustStorePassword="+trustStorePass,
		)
	}

	ctx, cancel := context.WithTimeout(e.Ctx, e.Platform.Status.Build.GetTimeout().Duration)
	defer cancel()

	if err := project.Command(mc).DoSettings(ctx); err != nil {
		return err
	}
	if err := project.Command(mc).DoPom(ctx); err != nil {
		return err
	}
	if err := project.Command(mc).Do(ctx); err != nil {
		return err
	}

	content, err = util.ReadFile(out)
	if err != nil {
		return err
	}

	if resource.Compression {
		c, err := gzip.CompressBase64(content)
		if err != nil {
			return err
		}

		content = c
	}

	// Compute the input digest and store it along with the configmap
	hash, err := digest.ComputeForResource(resource)
	if err != nil {
		return err
	}

	// Store the generated rest xml in a separate config map in order
	// not to pollute the integration with generated data
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      generatedContentName,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
			Annotations: map[string]string{
				sourceLanguageAnnotation:            string(v1.LanguageXML),
				sourceNameAnnotation:                resource.Name,
				sourceCompressionAnnotation:         strconv.FormatBool(resource.Compression),
				"camel.apache.org/source.generated": boolean.TrueString,
				"camel.apache.org/source.type":      "openapi",
				"camel.apache.org/source.digest":    hash,
			},
		},
		Data: map[string]string{
			"content": string(content),
		},
	}

	e.Resources.Add(&cm)
	return nil
}

func (t *openAPITrait) generateMavenProject(runtimeVersion string) maven.Project {
	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-rest-dsl-generator", defaults.Version)
	p.Build = &maven.Build{
		DefaultGoal: "generate-resources",
		Plugins: []maven.Plugin{
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-maven-plugin",
				Version:    runtimeVersion,
				Executions: []maven.Execution{
					{
						Phase: "generate-resources",
						Goals: []string{
							"generate-rest-xml",
						},
					},
				},
			},
		},
	}

	return p
}
