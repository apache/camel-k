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

package olm

import (
	"context"
	"strings"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "sigs.k8s.io/controller-runtime/pkg/client"
)

// The following properties can be overridden at build time via ldflags

// DefaultOperatorName is the Camel K operator name in OLM
var DefaultOperatorName = "camel-k-operator"

// DefaultPackage is the Camel K package in OLM
var DefaultPackage = "camel-k"

// DefaultChannel is the distribution channel in Operator Hub
var DefaultChannel = "alpha"

// DefaultSource is the name of the operator source where the operator is published
var DefaultSource = "community-operators"

// DefaultSourceNamespace is the namespace of the operator source
var DefaultSourceNamespace = "openshift-marketplace"

// DefaultStartingCSV contains the specific version to install
var DefaultStartingCSV = ""

// DefaultGlobalNamespace indicates a namespace containing an OperatorGroup that enables the operator to watch all namespaces.
// It will be used in global installation mode.
var DefaultGlobalNamespace = "openshift-operators"

// Options contains information about an operator in OLM
type Options struct {
	OperatorName        string
	Package             string
	Channel             string
	Source              string
	SourceNamespace     string
	StartingCSV         string
	GlobalNamespace     string
}

// IsOperatorInstalled tells if a OLM CSV or a Subscription is already installed in the namespace
func IsOperatorInstalled(ctx context.Context, client client.Client, namespace string, global bool, options Options) (bool, error) {
	options = fillDefaults(options)
	// CSV is present in current namespace for both local and global installation modes
	if csv, err := findCSV(ctx, client, namespace, options); err != nil {
		return false, err
	} else if csv != nil {
		return true, nil
	}
	// A subscription may indicate an in-progress installation
	if sub, err := findSubscription(ctx, client, namespace, global, options); err != nil {
		return false, err
	} else if sub != nil {
		return true, nil
	}

	return false, nil
}

// Install creates a subscription for the OLM package
func Install(ctx context.Context, client client.Client, namespace string, global bool, options Options, collection *kubernetes.Collection) error {
	options = fillDefaults(options)
	if installed, err := IsOperatorInstalled(ctx, client, namespace, global, options); err != nil {
		return err
	} else if installed {
		// Already installed
		return nil
	}

	targetNamespace := namespace
	if global {
		targetNamespace = options.GlobalNamespace
	}

	sub := olmv1alpha1.Subscription{
		ObjectMeta: v1.ObjectMeta{
			Name:      options.Package,
			Namespace: targetNamespace,
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			CatalogSource:          options.Source,
			CatalogSourceNamespace: options.SourceNamespace,
			Package:                options.Package,
			Channel:                options.Channel,
			StartingCSV:            options.StartingCSV,
			InstallPlanApproval:    olmv1alpha1.ApprovalAutomatic,
		},
	}
	if collection != nil {
		collection.Add(&sub)
		return nil
	}
	return client.Create(ctx, &sub)
}

// Uninstall removes CSV and subscription from the namespace
func Uninstall(ctx context.Context, client client.Client, namespace string, global bool, options Options) error {
	sub, err := findSubscription(ctx, client, namespace, global, options)
	if err != nil {
		return err
	}
	if sub != nil {
		if err := client.Delete(ctx, sub); err != nil {
			return err
		}
	}

	csv, err := findCSV(ctx, client, namespace, options)
	if err != nil {
		return err
	}
	if csv != nil {
		if err := client.Delete(ctx, csv); err != nil {
			return err
		}
	}
	return nil
}

func findSubscription(ctx context.Context, client client.Client, namespace string, global bool, options Options) (*olmv1alpha1.Subscription, error) {
	subNamespace := namespace
	if global {
		// In case of global installation, global subscription must be removed
		subNamespace = options.GlobalNamespace
	}
	subscriptionList := olmv1alpha1.SubscriptionList{}
	if err := client.List(ctx, &subscriptionList, runtime.InNamespace(subNamespace)); err != nil {
		return nil, err
	}

	for _, item := range subscriptionList.Items {
		if item.Spec.Package == options.Package {
			return &item, nil
		}
	}
	return nil, nil
}

func findCSV(ctx context.Context, client client.Client, namespace string, options Options) (*olmv1alpha1.ClusterServiceVersion, error) {
	csvList := olmv1alpha1.ClusterServiceVersionList{}
	if err := client.List(ctx, &csvList, runtime.InNamespace(namespace)); err != nil {
		return nil, err
	}

	for _, item := range csvList.Items {
		if strings.HasPrefix(item.Name, options.OperatorName) {
			return &item, nil
		}
	}
	return nil, nil
}

func fillDefaults(o Options) Options {
	if o.OperatorName == "" {
		o.OperatorName = DefaultOperatorName
	}
	if o.Package == "" {
		o.Package = DefaultPackage
	}
	if o.Channel == "" {
		o.Channel = DefaultChannel
	}
	if o.Source == "" {
		o.Source = DefaultSource
	}
	if o.SourceNamespace == "" {
		o.SourceNamespace = DefaultSourceNamespace
	}
	if o.StartingCSV == "" {
		o.StartingCSV = DefaultStartingCSV
	}
	if o.GlobalNamespace == "" {
		o.GlobalNamespace = DefaultGlobalNamespace
	}
	return o
}
