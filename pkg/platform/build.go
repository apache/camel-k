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

package platform

import (
	"context"
	"errors"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/build/assemble"
	"github.com/apache/camel-k/pkg/build/publish"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildManager is the current build manager
// Note: it cannot be changed at runtime, needs a operator restart
var buildManager *build.Manager

// GetPlatformBuildManager returns a suitable build manager for the current platform
func GetPlatformBuildManager(ctx context.Context, namespace string) (*build.Manager, error) {
	if buildManager != nil {
		return buildManager, nil
	}
	pl, err := GetCurrentPlatform(namespace)
	if err != nil {
		return nil, err
	}

	assembler := assemble.NewMavenAssembler(ctx)
	if pl.Spec.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyS2I {
		publisher := publish.NewS2IIncrementalPublisher(ctx, namespace, newContextLister(namespace))
		buildManager = build.NewManager(ctx, assembler, publisher)
	}

	if buildManager == nil {
		return nil, errors.New("unsupported platform configuration")
	}
	return buildManager, nil
}

// =================================================================

type contextLister struct {
	namespace string
}

func newContextLister(namespace string) contextLister {
	return contextLister{
		namespace: namespace,
	}
}

func (l contextLister) ListPublishedImages() ([]publish.PublishedImage, error) {
	list := v1alpha1.NewIntegrationContextList()

	err := sdk.List(l.namespace, &list, sdk.WithListOptions(&metav1.ListOptions{}))
	if err != nil {
		return nil, err
	}
	images := make([]publish.PublishedImage, 0)
	for _, ctx := range list.Items {
		if ctx.Status.Phase != v1alpha1.IntegrationContextPhaseReady || ctx.Labels == nil {
			continue
		}
		if ctxType, present := ctx.Labels["camel.apache.org/context.type"]; !present || ctxType != "platform" {
			continue
		}

		images = append(images, publish.PublishedImage{
			Image:     ctx.Status.Image,
			Classpath: ctx.Status.Classpath,
		})
	}
	return images, nil
}
