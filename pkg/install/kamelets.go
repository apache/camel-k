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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

const kameletDirEnv = "KAMELET_CATALOG_DIR"
const defaultKameletDir = "/kamelets/"

// KameletCatalog installs the bundled KameletCatalog into one namespace
func KameletCatalog(ctx context.Context, c client.Client, namespace string) error {
	kameletDir := os.Getenv(kameletDirEnv)
	if kameletDir == "" {
		kameletDir = defaultKameletDir
	}
	if d, err := os.Stat(kameletDir); err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	} else if !d.IsDir() {
		return fmt.Errorf("Kamelet directory %q is a file", kameletDir)
	}

	files, err := ioutil.ReadDir(kameletDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || !(strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml")) {
			continue
		}

		content, err := ioutil.ReadFile(path.Join(kameletDir, file.Name()))
		if err != nil {
			return err
		}

		obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), string(content))
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
				if k.GetAnnotations() == nil {
					k.SetAnnotations(make(map[string]string))
				}
				k.GetAnnotations()[kamelVersionAnnotation] = defaults.Version

				if k.GetLabels() == nil {
					k.SetLabels(make(map[string]string))
				}
				k.GetLabels()[v1alpha1.KameletBundledLabel] = "true"
				k.GetLabels()[v1alpha1.KameletReadOnlyLabel] = "true"

				err := ObjectOrCollect(ctx, c, namespace, nil, true, k)

				if err != nil {
					return errors.Wrapf(err, "could not create resource from file %q", path.Join(kameletDir, file.Name()))
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
