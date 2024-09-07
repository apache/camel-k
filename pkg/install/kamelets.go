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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

const (
	kameletDirEnv     = "KAMELET_CATALOG_DIR"
	defaultKameletDir = "/kamelets/"
)

// KameletCatalog installs the bundled Kamelets into the specified namespace.
func KameletCatalog(ctx context.Context, c client.Client, namespace string) error {
	kameletDir := os.Getenv(kameletDirEnv)
	if kameletDir == "" {
		kameletDir = defaultKameletDir
	}
	d, err := os.Stat(kameletDir)
	switch {
	case err != nil && os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	case !d.IsDir():
		return fmt.Errorf("kamelet directory %q is a file", kameletDir)
	}

	g, gCtx := errgroup.WithContext(ctx)
	applier := c.ServerOrClientSideApplier()

	err = filepath.WalkDir(kameletDir, func(p string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() && f.Name() != d.Name() {
			return fs.SkipDir
		}
		if !(strings.HasSuffix(f.Name(), ".yaml") || strings.HasSuffix(f.Name(), ".yml")) {
			return nil
		}
		// We may want to throttle the creation of Go routines if the number of bundled Kamelets increases.
		g.Go(func() error {
			kamelet, err := loadKamelet(filepath.Join(kameletDir, f.Name()), namespace)
			if err != nil {
				return err
			}
			err = applier.Apply(gCtx, kamelet)
			// We only log the error. If we returned the error, the creation of the ITP would have stopped
			if err != nil {
				log.Error(err, "Error occurred whilst applying bundled kamelet")
			}
			return nil
		})
		return nil
	})
	if err != nil {
		return err
	}

	return g.Wait()
}

func loadKamelet(path string, namespace string) (ctrl.Object, error) {
	content, err := util.ReadFile(path)
	if err != nil {
		return nil, err
	}

	kamelet, err := kubernetes.LoadUnstructuredFromYaml(string(content))
	if err != nil {
		return nil, err
	}
	gvk := kamelet.GetObjectKind().GroupVersionKind()
	if gvk.Group != v1.SchemeGroupVersion.Group || gvk.Kind != "Kamelet" {
		return nil, fmt.Errorf("file %q does not define a Kamelet", path)
	}

	kamelet.SetNamespace(namespace)

	annotations := kamelet.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[kamelVersionAnnotation] = defaults.Version
	kamelet.SetAnnotations(annotations)

	labels := kamelet.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[v1.KameletBundledLabel] = "true"
	labels[v1.KameletReadOnlyLabel] = "true"

	kamelet.SetLabels(labels)

	return kamelet, nil
}

// KameletViewerRole installs the role that allows any user ro access kamelets in the global namespace.
func KameletViewerRole(ctx context.Context, c client.Client, namespace string) error {
	return Resource(ctx, c, namespace, true, IdentityResourceCustomizer, "/resources/viewer/user-global-kamelet-viewer-role-binding.yaml")
}
