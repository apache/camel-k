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

// Package s2i contains utilities for openshift s2i builds.
package s2i

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/apache/camel-k/v2/pkg/client"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Cancel the s2i Build by updating its status.
func CancelBuild(ctx context.Context, c client.Client, build *buildv1.Build) error {
	target := build.DeepCopy()
	target.Status.Cancelled = true
	if err := c.Patch(ctx, target, ctrl.MergeFrom(build)); err != nil {
		return err
	}
	*build = *target
	return nil
}

// Wait for the s2i Build to complete with success or cancellation.
func WaitForS2iBuildCompletion(ctx context.Context, c client.Client, build *buildv1.Build) error {
	key := ctrl.ObjectKeyFromObject(build)
	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(1 * time.Second):
			err := c.Get(ctx, key, build)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return err
			}

			if build.Status.Phase == buildv1.BuildPhaseComplete {
				return nil
			} else if build.Status.Phase == buildv1.BuildPhaseCancelled ||
				build.Status.Phase == buildv1.BuildPhaseFailed ||
				build.Status.Phase == buildv1.BuildPhaseError {
				return errors.New("build failed")
			}
		}
	}
}

// Create the BuildConfig of the build with the right owner after having deleted it already existed.
func BuildConfig(ctx context.Context, c client.Client, bc *buildv1.BuildConfig, owner metav1.Object) error {
	if err := c.Delete(ctx, bc); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("cannot delete build config: %w", err)
	}

	if err := ctrlutil.SetOwnerReference(owner, bc, c.GetScheme()); err != nil {
		return fmt.Errorf("cannot set owner reference on BuildConfig: %s: %w", bc.Name, err)
	}

	if err := c.Create(ctx, bc); err != nil {
		return fmt.Errorf("cannot create build config: %w", err)
	}
	return nil
}

// Create the ImageStream for the builded image with the right owner after having deleted it already existed.
func ImageStream(ctx context.Context, c client.Client, is *imagev1.ImageStream, owner metav1.Object) error {
	if err := c.Delete(ctx, is); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("cannot delete image stream: %w", err)
	}

	if err := ctrlutil.SetOwnerReference(owner, is, c.GetScheme()); err != nil {
		return fmt.Errorf("cannot set owner reference on ImageStream: %s: %w", is.Name, err)
	}

	if err := c.Create(ctx, is); err != nil {
		return fmt.Errorf("cannot create image stream: %w", err)
	}
	return nil
}
