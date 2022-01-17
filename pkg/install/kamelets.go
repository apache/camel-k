/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at

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
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/patch"
)

const (
	kameletDirEnv     = "KAMELET_CATALOG_DIR"
	defaultKameletDir = "/kamelets/"
)

var (
	log = logf.Log

	hasServerSideApply atomic.Value
	tryServerSideApply sync.Once
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
			kamelet, err := loadKamelet(path.Join(kameletDir, f.Name()), namespace)
			if err != nil {
				return err
			}
			once := false
			tryServerSideApply.Do(func() {
				once = true
				if err = serverSideApply(gCtx, c, kamelet); err != nil {
					if isIncompatibleServerError(err) {
						log.Info("Fallback to client-side apply for installing bundled Kamelets")
						hasServerSideApply.Store(false)
						err = nil
					} else {
						tryServerSideApply = sync.Once{}
					}
				} else {
					hasServerSideApply.Store(true)
				}
			})
			if err != nil {
				return err
			}
			if v := hasServerSideApply.Load(); v.(bool) {
				if !once {
					return serverSideApply(gCtx, c, kamelet)
				}
			} else {
				return clientSideApply(gCtx, c, kamelet)
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

func serverSideApply(ctx context.Context, c client.Client, resource runtime.Object) error {
	target, err := patch.ApplyPatch(resource)
	if err != nil {
		return err
	}
	return c.Patch(ctx, target, ctrl.Apply, ctrl.ForceOwnership, ctrl.FieldOwner("camel-k-operator"))
}

func clientSideApply(ctx context.Context, c client.Client, resource ctrl.Object) error {
	err := c.Create(ctx, resource)
	if err == nil {
		return nil
	} else if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("error during create resource: %s/%s: %w", resource.GetNamespace(), resource.GetName(), err)
	}
	object := &unstructured.Unstructured{}
	object.SetNamespace(resource.GetNamespace())
	object.SetName(resource.GetName())
	object.SetGroupVersionKind(resource.GetObjectKind().GroupVersionKind())
	err = c.Get(ctx, ctrl.ObjectKeyFromObject(object), object)
	if err != nil {
		return err
	}
	p, err := patch.MergePatch(object, resource)
	if err != nil {
		return err
	} else if len(p) == 0 {
		return nil
	}
	return c.Patch(ctx, resource, ctrl.RawPatch(types.MergePatchType, p))
}

func isIncompatibleServerError(err error) bool {
	// First simpler check for older servers (i.e. OpenShift 3.11)
	if strings.Contains(err.Error(), "415: Unsupported Media Type") {
		return true
	}
	// 415: Unsupported media type means we're talking to a server which doesn't
	// support server-side apply.
	var serr *k8serrors.StatusError
	if errors.As(err, &serr) {
		return serr.Status().Code == http.StatusUnsupportedMediaType
	}
	// Non-StatusError means the error isn't because the server is incompatible.
	return false
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
	if gvk.Group != v1alpha1.SchemeGroupVersion.Group || gvk.Kind != "Kamelet" {
		return nil, fmt.Errorf("file %q does not define a Kamelet", path)
	}

	kamelet.SetNamespace(namespace)

	if kamelet.GetAnnotations() == nil {
		kamelet.SetAnnotations(make(map[string]string))
	}
	kamelet.GetAnnotations()[kamelVersionAnnotation] = defaults.Version

	if kamelet.GetLabels() == nil {
		kamelet.SetLabels(make(map[string]string))
	}
	kamelet.GetLabels()[v1alpha1.KameletBundledLabel] = "true"
	kamelet.GetLabels()[v1alpha1.KameletReadOnlyLabel] = "true"

	return kamelet, nil
}

// KameletViewerRole installs the role that allows any user ro access kamelets in the global namespace.
func KameletViewerRole(ctx context.Context, c client.Client, namespace string) error {
	if err := Resource(ctx, c, namespace, true, IdentityResourceCustomizer, "/viewer/user-global-kamelet-viewer-role.yaml"); err != nil {
		return err
	}
	return Resource(ctx, c, namespace, true, IdentityResourceCustomizer, "/viewer/user-global-kamelet-viewer-role-binding.yaml")
}
