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

package catalog

import (
	"context"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewInitializeAction returns a action that initializes the catalog configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(catalog *v1.CamelCatalog) bool {
	return catalog.Status.Phase == v1.CamelCatalogPhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	action.L.Info("Initializing CamelCatalog")
	target := catalog.DeepCopy()

	if catalog.Spec.GetQuarkusToolingImage() == "" {
		target.Status.Phase = v1.CamelCatalogPhaseError
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Container image tool",
			"Container image tool missing in catalog. This catalog is not compatible with Camel K version above 2.0",
		)
	} else {
		target.Status.Phase = v1.CamelCatalogPhaseReady
		target.Status.SetCondition(
			v1.CamelCatalogConditionReady,
			corev1.ConditionTrue,
			"Container image tool",
			"Container image tool found in catalog",
		)
	}

	action.L.Info("Cloning CamelCatalog for Plain Quarkus runtime")
	if err := action.addPlainQuarkusCatalog(ctx, catalog); err != nil {
		// Only warn the user, we don't want to fail
		action.L.Infof(
			"WARN: the operator wasn't able to clone %s catalog. You won't be able to run a plain Quarkus runtime provider.",
			catalog.Name,
		)
	}

	return target, nil
}

// addPlainQuarkusCatalog is a workaround while a CamelCatalog custom resource is required. The goal is to clone any existing
// Camel K Runtime catalog and adjust to make it work with plain Quarkus runtime provider.
func (action *initializeAction) addPlainQuarkusCatalog(ctx context.Context, catalog *v1.CamelCatalog) error {
	runtimeSpec := v1.RuntimeSpec{
		Version:  catalog.Spec.GetRuntimeVersion(),
		Provider: v1.RuntimeProviderPlainQuarkus,
	}
	cat, err := loadCatalog(ctx, action.client, catalog.Namespace, runtimeSpec)
	if err != nil {
		return err
	}
	if cat == nil {
		// Clone the catalog to enable Quarkus Plain runtime
		clonedCatalog := catalog.DeepCopy()
		clonedCatalog.Status = v1.CamelCatalogStatus{}
		clonedCatalog.ObjectMeta = metav1.ObjectMeta{
			Namespace:   catalog.Namespace,
			Name:        strings.ReplaceAll(catalog.Name, "camel-catalog", "camel-catalog-quarkus"),
			Labels:      catalog.Labels,
			Annotations: catalog.Annotations,
		}
		clonedCatalog.Spec.Runtime.Provider = v1.RuntimeProviderPlainQuarkus
		clonedCatalog.Spec.Runtime.Dependencies = []v1.MavenArtifact{
			{
				GroupID:    v1.MavenQuarkusGroupID,
				ArtifactID: "camel-quarkus-core",
			},
			// We enforce the presence of this component to provide an
			// opinionated set of observability services
			{
				GroupID:    v1.MavenQuarkusGroupID,
				ArtifactID: "camel-quarkus-observability-services",
			},
		}
		if clonedCatalog.Spec.Runtime.Capabilities != nil {
			clonedCatalog.Spec.Runtime.Capabilities["cron"] = v1.Capability{
				Dependencies: []v1.MavenArtifact{},
			}
			clonedCatalog.Spec.Runtime.Capabilities["knative"] = v1.Capability{
				Dependencies: []v1.MavenArtifact{
					{
						GroupID:    v1.MavenQuarkusGroupID,
						ArtifactID: "camel-quarkus-knative",
					},
				},
			}
			runtimesProps := clonedCatalog.Spec.Runtime.Capabilities["master"].RuntimeProperties
			clonedCatalog.Spec.Runtime.Capabilities["master"] = v1.Capability{
				Dependencies: []v1.MavenArtifact{
					{
						GroupID:    v1.MavenQuarkusGroupID,
						ArtifactID: "camel-quarkus-master",
					},
					{
						GroupID:    v1.MavenQuarkusGroupID,
						ArtifactID: "camel-quarkus-kubernetes",
					},
					{
						GroupID:    v1.MavenQuarkusGroupID,
						ArtifactID: "camel-quarkus-kubernetes-cluster-service",
					},
				},
				RuntimeProperties: runtimesProps,
			}
			clonedCatalog.Spec.Runtime.Capabilities["resume-kafka"] = v1.Capability{
				Dependencies: []v1.MavenArtifact{},
			}
			clonedCatalog.Spec.Runtime.Capabilities["jolokia"] = v1.Capability{
				Dependencies: []v1.MavenArtifact{
					{
						GroupID:    v1.MavenQuarkusGroupID,
						ArtifactID: "camel-quarkus-jaxb",
					},
					{
						GroupID:    v1.MavenQuarkusGroupID,
						ArtifactID: "camel-quarkus-management",
					},
					{
						GroupID:    "org.jolokia",
						ArtifactID: "jolokia-agent-jvm",
						Classifier: "javaagent",
						Version:    "2.1.1",
					},
				},
			}
		}

		return action.client.Create(ctx, clonedCatalog)
	}

	return nil
}

func loadCatalog(ctx context.Context, c client.Client, namespace string, runtimeSpec v1.RuntimeSpec) (*v1.CamelCatalog, error) {
	options := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	list := v1.NewCamelCatalogList()
	if err := c.List(ctx, &list, options...); err != nil {
		return nil, err
	}
	for _, cc := range list.Items {
		if cc.Spec.Runtime.Provider == runtimeSpec.Provider && cc.Spec.Runtime.Version == runtimeSpec.Version {
			return &cc, nil
		}
	}

	return nil, nil
}
