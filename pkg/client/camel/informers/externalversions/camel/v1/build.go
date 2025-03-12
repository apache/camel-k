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

package v1

import (
	context "context"
	time "time"

	apiscamelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	versioned "github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned"
	internalinterfaces "github.com/apache/camel-k/v2/pkg/client/camel/informers/externalversions/internalinterfaces"
	camelv1 "github.com/apache/camel-k/v2/pkg/client/camel/listers/camel/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// BuildInformer provides access to a shared informer and lister for
// Builds.
type BuildInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() camelv1.BuildLister
}

type buildInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewBuildInformer constructs a new informer for Build type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBuildInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBuildInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredBuildInformer constructs a new informer for Build type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBuildInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CamelV1().Builds(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CamelV1().Builds(namespace).Watch(context.TODO(), options)
			},
		},
		&apiscamelv1.Build{},
		resyncPeriod,
		indexers,
	)
}

func (f *buildInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBuildInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *buildInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apiscamelv1.Build{}, f.defaultInformer)
}

func (f *buildInformer) Lister() camelv1.BuildLister {
	return camelv1.NewBuildLister(f.Informer().GetIndexer())
}
