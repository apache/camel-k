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

package install

import (
	"context"
	"path"
	"strings"

	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/resources"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

const kameletDir = "/kamelets/"
const kameletBundledLabel = "camel.apache.org/kamelet.bundled"
const kameletReadOnlyLabel = "camel.apache.org/kamelet.readonly"

// KameletCatalog installs the bundled KameletCatalog into one namespace
func KameletCatalog(ctx context.Context, c client.Client, namespace string) error {
	if !resources.DirExists(kameletDir) {
		return nil
	}

	for _, res := range resources.Resources(kameletDir) {
		if !strings.HasSuffix(res, ".yaml") && !strings.HasSuffix(res, ".yml") {
			continue
		}

		obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), resources.ResourceAsString(path.Join(kameletDir, res)))
		if err != nil {
			return err
		}
		if k, ok := obj.(*v1alpha1.Kamelet); ok {
			existing := &v1alpha1.Kamelet{}
			err = c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: k.Name}, existing)
			if err != nil {
				if k8serrors.IsNotFound(err) {
					existing = nil
				} else {
					return err
				}
			}

			if existing == nil || existing.Annotations[kamelVersionAnnotation] != defaults.Version {
				err := Resource(ctx, c, namespace, true, func(o ctrl.Object) ctrl.Object {
					if o.GetAnnotations() == nil {
						o.SetAnnotations(make(map[string]string))
					}
					o.GetAnnotations()[kamelVersionAnnotation] = defaults.Version

					if o.GetLabels() == nil {
						o.SetLabels(make(map[string]string))
					}
					o.GetLabels()[kameletBundledLabel] = "true"
					o.GetLabels()[kameletReadOnlyLabel] = "true"
					return o
				}, path.Join(kameletDir, res))

				if err != nil {
					return errors.Wrapf(err, "could not create resource %q", res)
				}
			}
		}
	}

	return nil
}

// KameletViewerRole installs the role that allows any user ro access kamelets in the global namespace
func KameletViewerRole(ctx context.Context, c client.Client, namespace string) error {
	if err := Resource(ctx, c, namespace, true, IdentityResourceCustomizer, "/rbac/user-global-kamelet-viewer-role.yaml"); err != nil {
		return err
	}
	return Resource(ctx, c, namespace, true, IdentityResourceCustomizer, "/rbac/user-global-kamelet-viewer-role-binding.yaml")
}
