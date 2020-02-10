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
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	OperatorName    string
	Package         string
	Channel         string
	Source          string
	SourceNamespace string
	StartingCSV     string
	GlobalNamespace string
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

// HasPermissionToInstall checks if the current user/serviceaccount has the right permissions to install camel k via OLM
func HasPermissionToInstall(ctx context.Context, client client.Client, namespace string, global bool, options Options) (bool, error) {
	if ok, err := checkPermission(client, olmv1alpha1.GroupName, "clusterserviceversions", namespace, options.Package, "list"); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	targetNamespace := namespace
	if global {
		targetNamespace = options.GlobalNamespace
	}

	if ok, err := checkPermission(client, olmv1alpha1.GroupName, "subscriptions", targetNamespace, options.Package, "create"); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	if installed, err := IsOperatorInstalled(ctx, client, namespace, global, options); err != nil {
		return false, err
	} else if installed {
		return true, nil
	}

	if !global {
		if ok, err := checkPermission(client, olmv1.GroupName, "operatorgroups", namespace, options.Package, "list"); err != nil {
			return false, err
		} else if !ok {
			return false, nil
		}

		group, err := findOperatorGroup(ctx, client, namespace, options)
		if err != nil {
			return false, err
		}
		if group == nil {
			if ok, err := checkPermission(client, olmv1.GroupName, "operatorgroups", namespace, options.Package, "create"); err != nil {
				return false, err
			} else if !ok {
				return false, nil
			}
		}

	}
	return true, nil
}

// nolint:unparam
func checkPermission(client client.Client, group, resource, namespace, name, verb string) (bool, error) {
	sarReview := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Group:     group,
				Resource:  resource,
				Namespace: namespace,
				Name:      name,
				Verb:      verb,
			},
		},
	}

	sar, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(sarReview)
	if err != nil {
		if k8serrors.IsForbidden(err) {
			return false, nil
		}
		return false, err
	} else if !sar.Status.Allowed {
		return false, nil
	}
	return true, nil
}

// Install creates a subscription for the OLM package
func Install(ctx context.Context, client client.Client, namespace string, global bool, options Options, collection *kubernetes.Collection) (bool, error) {
	options = fillDefaults(options)
	if installed, err := IsOperatorInstalled(ctx, client, namespace, global, options); err != nil {
		return false, err
	} else if installed {
		// Already installed
		return false, nil
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
	} else if err := client.Create(ctx, &sub); err != nil {
		return false, err
	}

	if !global {
		group, err := findOperatorGroup(ctx, client, namespace, options)
		if err != nil {
			return false, err
		}
		if group == nil {
			group = &olmv1.OperatorGroup{
				ObjectMeta: v1.ObjectMeta{
					Namespace:    namespace,
					GenerateName: fmt.Sprintf("%s-", namespace),
				},
				Spec: olmv1.OperatorGroupSpec{
					TargetNamespaces: []string{namespace},
				},
			}
			if collection != nil {
				collection.Add(group)
			} else if err := client.Create(ctx, group); err != nil {
				return false, errors.Wrap(err, fmt.Sprintf("namespace %s has no operator group defined and "+
					"current user is not able to create it. "+
					"Make sure you have the right roles to install operators from OLM", namespace))
			}
		}
	}
	return true, nil
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

// nolint:unparam
func findOperatorGroup(ctx context.Context, client client.Client, namespace string, options Options) (*olmv1.OperatorGroup, error) {
	opGroupList := olmv1.OperatorGroupList{}
	if err := client.List(ctx, &opGroupList, runtime.InNamespace(namespace)); err != nil {
		return nil, err
	}

	if len(opGroupList.Items) > 0 {
		return &opGroupList.Items[0], nil
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
