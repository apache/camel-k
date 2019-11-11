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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	camelv1alpha1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	versioned "github.com/apache/camel-k/pkg/client/clientset/versioned"
	internalinterfaces "github.com/apache/camel-k/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/apache/camel-k/pkg/client/listers/camel/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// IntegrationPlatformInformer provides access to a shared informer and lister for
// IntegrationPlatforms.
type IntegrationPlatformInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.IntegrationPlatformLister
}

type integrationPlatformInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewIntegrationPlatformInformer constructs a new informer for IntegrationPlatform type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewIntegrationPlatformInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredIntegrationPlatformInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredIntegrationPlatformInformer constructs a new informer for IntegrationPlatform type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredIntegrationPlatformInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CamelV1alpha1().IntegrationPlatforms(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CamelV1alpha1().IntegrationPlatforms(namespace).Watch(options)
			},
		},
		&camelv1alpha1.IntegrationPlatform{},
		resyncPeriod,
		indexers,
	)
}

func (f *integrationPlatformInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredIntegrationPlatformInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *integrationPlatformInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&camelv1alpha1.IntegrationPlatform{}, f.defaultInformer)
}

func (f *integrationPlatformInformer) Lister() v1alpha1.IntegrationPlatformLister {
	return v1alpha1.NewIntegrationPlatformLister(f.Informer().GetIndexer())
}
