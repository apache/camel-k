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

package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/patch"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type ServerOrClientSideApplier struct {
	Client             ctrl.Client
	hasServerSideApply atomic.Value
	tryServerSideApply sync.Once
}

func (c *defaultClient) ServerOrClientSideApplier() ServerOrClientSideApplier {
	return ServerOrClientSideApplier{
		Client: c,
	}
}

func (a *ServerOrClientSideApplier) Apply(ctx context.Context, object ctrl.Object) error {
	once := false
	var err error
	a.tryServerSideApply.Do(func() {
		once = true
		if err = a.serverSideApply(ctx, object); err != nil {
			if isIncompatibleServerError(err) {
				log.Info("Fallback to client-side apply for installing resources")
				a.hasServerSideApply.Store(false)
				err = nil
			}
		} else {
			a.hasServerSideApply.Store(true)
		}
	})
	if err != nil {
		a.tryServerSideApply = sync.Once{}
		return err
	}
	if v := a.hasServerSideApply.Load(); v.(bool) {
		if !once {
			return a.serverSideApply(ctx, object)
		}
	} else {
		return a.clientSideApply(ctx, object)
	}
	return nil
}

func (a *ServerOrClientSideApplier) serverSideApply(ctx context.Context, resource runtime.Object) error {
	target, err := patch.ApplyPatch(resource)
	if err != nil {
		return err
	}
	return a.Client.Patch(ctx, target, ctrl.Apply, ctrl.ForceOwnership, ctrl.FieldOwner("camel-k-operator"))
}

func (a *ServerOrClientSideApplier) clientSideApply(ctx context.Context, resource ctrl.Object) error {
	err := a.Client.Create(ctx, resource)
	if err == nil {
		return nil
	} else if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("error during create resource: %s/%s: %w", resource.GetNamespace(), resource.GetName(), err)
	}
	object := &unstructured.Unstructured{}
	object.SetNamespace(resource.GetNamespace())
	object.SetName(resource.GetName())
	object.SetGroupVersionKind(resource.GetObjectKind().GroupVersionKind())
	err = a.Client.Get(ctx, ctrl.ObjectKeyFromObject(object), object)
	if err != nil {
		return err
	}
	p, err := patch.MergePatch(object, resource)
	if err != nil {
		return err
	} else if len(p) == 0 {
		return nil
	}
	return a.Client.Patch(ctx, resource, ctrl.RawPatch(types.MergePatchType, p))
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
