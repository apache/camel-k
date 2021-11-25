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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/multierr"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/gzip"
	"github.com/apache/camel-k/pkg/util/jvm"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/maven"
)

// The OpenAPI DSL trait is internally used to allow creating integrations from a OpenAPI specs.
//
// +camel-k:trait=openapi.
type openAPITrait struct {
	BaseTrait `property:",squash"`
}

func newOpenAPITrait() Trait {
	return &openAPITrait{
		BaseTrait: NewBaseTrait("openapi", 300),
	}
}

// IsPlatformTrait overrides base class method.
func (t *openAPITrait) IsPlatformTrait() bool {
	return true
}

func (t *openAPITrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	if e.Integration == nil {
		return false, nil
	}

	// check if the runtime provides 'rest' capabilities
	if _, ok := e.CamelCatalog.Runtime.Capabilities[v1.CapabilityRest]; !ok {
		return false, fmt.Errorf("the runtime provider %s does not declare 'rest' capability", e.CamelCatalog.Runtime.Provider)
	}

	for _, resource := range e.Integration.Spec.Resources {
		if resource.Type == v1.ResourceTypeOpenAPI {
			return e.IntegrationInPhase(v1.IntegrationPhaseInitialization), nil
		}
	}

	return false, nil
}

func (t *openAPITrait) Apply(e *Environment) error {
	util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityRest)

	root := os.TempDir()
	tmpDir, err := ioutil.TempDir(root, "openapi")
	if err != nil {
		return err
	}

	for i, resource := range e.Integration.Spec.Resources {
		if resource.Type != v1.ResourceTypeOpenAPI {
			continue
		}
		if resource.Name == "" {
			return multierr.Append(
				fmt.Errorf("no name defined for the openapi resource: %v", resource),
				os.RemoveAll(tmpDir))
		}

		generatedContentName := fmt.Sprintf("%s-openapi-%03d", e.Integration.Name, i)

		// Generate configmap or reuse existing one
		if err := t.generateOpenAPIConfigMap(e, resource, tmpDir, generatedContentName); err != nil {
			return errors.Wrapf(err, "cannot generate configmap for openapi resource %s", resource.Name)
		}

		generatedSourceName := strings.TrimSuffix(resource.Name, filepath.Ext(resource.Name)) + ".xml"
		generatedSources := make([]v1.SourceSpec, 0, len(e.Integration.Status.GeneratedSources))

		if e.Integration.Status.GeneratedSources != nil {
			// Filter out the previously generated source
			for _, x := range e.Integration.Status.GeneratedSources {
				if x.Name != generatedSourceName {
					generatedSources = append(generatedSources, x)
				}
			}
		}

		// Add an additional source that references the config map
		generatedSources = append(generatedSources, v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:        generatedSourceName,
				ContentRef:  generatedContentName,
				Compression: resource.Compression,
			},
			Language: v1.LanguageXML,
		})

		e.Integration.Status.GeneratedSources = generatedSources
	}

	return os.RemoveAll(tmpDir)
}

func (t *openAPITrait) generateOpenAPIConfigMap(e *Environment, resource v1.ResourceSpec, tmpDir, generatedContentName string) error {
	cm := corev1.ConfigMap{}
	key := client.ObjectKey{
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

func (t *openAPITrait) createNewOpenAPIConfigMap(e *Environment, resource v1.ResourceSpec, tmpDir, generatedContentName string) error {
	tmpDir = path.Join(tmpDir, generatedContentName)
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

	in := path.Join(tmpDir, resource.Name)
	out := path.Join(tmpDir, "openapi-dsl.xml")

	err = ioutil.WriteFile(in, content, 0o400)
	if err != nil {
		return err
	}

	project, err := t.generateMavenProject(e)
	if err != nil {
		return err
	}

	mc := maven.NewContext(tmpDir)
	mc.LocalRepository = e.Platform.Status.Build.Maven.LocalRepository
	mc.AddArgument("-Dopenapi.spec=" + in)
	mc.AddArgument("-Ddsl.out=" + out)

	settings, err := kubernetes.ResolveValueSource(e.Ctx, e.Client, e.Platform.Namespace, &e.Platform.Status.Build.Maven.Settings)
	if err != nil {
		return err
	}
	if settings != "" {
		mc.SettingsContent = []byte(settings)
	}

	if e.Platform.Status.Build.Maven.CASecret != nil {
		certData, err := kubernetes.GetSecretRefData(e.Ctx, e.Client, e.Platform.Namespace, e.Platform.Status.Build.Maven.CASecret)
		if err != nil {
			return err
		}
		trustStoreName := "trust.jks"
		trustStorePass := jvm.NewKeystorePassword()
		err = jvm.GenerateKeystore(e.Ctx, tmpDir, trustStoreName, trustStorePass, certData)
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
	err = project.Command(mc).Do(ctx)
	if err != nil {
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
				"camel.apache.org/source.language":    string(v1.LanguageXML),
				"camel.apache.org/source.name":        resource.Name,
				"camel.apache.org/source.compression": strconv.FormatBool(resource.Compression),
				"camel.apache.org/source.generated":   "true",
				"camel.apache.org/source.type":        string(v1.ResourceTypeOpenAPI),
				"camel.apache.org/source.digest":      hash,
			},
		},
		Data: map[string]string{
			"content": string(content),
		},
	}

	e.Resources.Add(&cm)
	return nil
}

func (t *openAPITrait) generateMavenProject(e *Environment) (maven.Project, error) {
	if e.CamelCatalog == nil {
		return maven.Project{}, errors.New("unknown camel catalog")
	}

	p := maven.NewProjectWithGAV("org.apache.camel.k.integration", "camel-k-rest-dsl-generator", defaults.Version)
	p.Build = &maven.Build{
		DefaultGoal: "generate-resources",
		Plugins: []maven.Plugin{
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-maven-plugin",
				Version:    e.CamelCatalog.Runtime.Version,
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

	return p, nil
}
