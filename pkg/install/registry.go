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
	"os"
	"os/exec"
	"time"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

// OperatorStartupRegistry tries to install optional development container registry.
func OperatorStartupRegistry(ctx context.Context, c client.Client, withSecret bool) error {
	secret, err := resources.Resource("/resources/registry/secret.yaml")
	if err != nil {
		return fmt.Errorf("could not load development container registry Secret configuration", err)
	}
	var registrySecret corev1.Secret
	if err = yaml.Unmarshal(secret, &registrySecret); err != nil {
		return fmt.Errorf("could not parse development container registry Secret configuration", err)
	}

	service, err := resources.Resource("/resources/registry/service.yaml")
	if err != nil {
		return fmt.Errorf("could not load development container registry Service configuration", err)
	}
	var registryService corev1.Service
	if err = yaml.Unmarshal(service, &registryService); err != nil {
		return fmt.Errorf("could not parse development container registry Service configuration", err)
	}

	deploy, err := resources.Resource("/resources/registry/deploy.yaml")
	if err != nil {
		return fmt.Errorf("could not load development container registry Deployment configuration", err)
	}
	var registryDeploy appsv1.Deployment
	if err = yaml.Unmarshal(deploy, &registryDeploy); err != nil {
		return fmt.Errorf("could not parse development container registry Deployment configuration", err)
	}

	if withSecret && len(registryDeploy.Spec.Template.Spec.Containers) > 0 {
		registryDeploy.Spec.Template.Spec.Containers[0].Env =
			append(registryDeploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  "REGISTRY_AUTH",
				Value: "htpasswd",
			})
		registryDeploy.Spec.Template.Spec.Containers[0].Env =
			append(registryDeploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  "REGISTRY_AUTH_HTPASSWD_REALM",
				Value: "Registry Realm",
			})
		registryDeploy.Spec.Template.Spec.Containers[0].Env =
			append(registryDeploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  "REGISTRY_AUTH_HTPASSWD_PATH",
				Value: "/auth/htpasswd",
			})
	}

	// Get owner reference to operator deployment to manage garbage collection
	ref, err := getOwnerRef(ctx, c)
	if err != nil {
		return fmt.Errorf("could not get operator deployment ownership", err)
	}

	// create self signed certificate
	cmd := exec.Command("openssl",
		"req", "-x509", "-newkey", "rsa:2048", "-nodes",
		"-keyout", "/tmp/registry.key",
		"-out", "/tmp/registry.crt",
		"-days", "365",
		"-subj", "/CN=registry",
	)

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("could not generate development container registry self signed certificate", err)
	}
	crtSecret, err := kubernetes.TLSSecretFromFiles(ctx, "camel-k", "registry-tls", "/tmp/registry.crt", "/tmp/registry.key")
	if err != nil {
		return fmt.Errorf("could not generate development container registry self signed certificate secret", err)
	}
	crtSecret.SetOwnerReferences([]metav1.OwnerReference{*ref})
	if err := c.Create(ctx, crtSecret); err != nil {
		return fmt.Errorf("could not create development container registry push secret", err)
	}

	registrySecret.SetNamespace("camel-k")
	registryService.SetNamespace("camel-k")
	registryDeploy.SetNamespace("camel-k")
	registrySecret.SetOwnerReferences([]metav1.OwnerReference{*ref})
	registryService.SetOwnerReferences([]metav1.OwnerReference{*ref})
	registryDeploy.SetOwnerReferences([]metav1.OwnerReference{*ref})

	// Try to create the resources now
	if err := c.Create(ctx, &registrySecret); err != nil {
		return fmt.Errorf("could not create development container registry Secret configuration", err)
	}
	if err := c.Create(ctx, &registryService); err != nil {
		return fmt.Errorf("could not create development container registry Service configuration", err)
	}
	if err := c.Create(ctx, &registryDeploy); err != nil {
		return fmt.Errorf("could not create development container registry Deployment configuration", err)
	}

	// Get the cluster IP and use it to configure internally the operator
	clusterIP, err := waitForClusterIP(ctx, c, registryService.GetNamespace(), registryService.GetName(), 30*time.Second)
	if err != nil {
		return fmt.Errorf("could not get development container registry Service IP", err)
	}

	dockerRegistrySecret, err := kubernetes.DockerRegistrySecret(ctx, "camel-k", "ck-dev-registry", clusterIP, "admin", "password")
	if err != nil {
		return fmt.Errorf("could not generate development container registry push secret", err)
	}
	dockerRegistrySecret.SetOwnerReferences([]metav1.OwnerReference{*ref})
	if err := c.Create(ctx, dockerRegistrySecret); err != nil {
		return fmt.Errorf("could not create development container registry push secret", err)
	}

	log.Infof("Setting up development container registry configuration environment variables (registry IP %s). Notice that it overrides the operator configuration"+
		" but it won't override any IntegrationProfile configuration.", clusterIP)
	os.Setenv("REGISTRY_ADDRESS", clusterIP)
	os.Setenv("REGISTRY_INSECURE", "false")
	os.Setenv("REGISTRY_SECRET", dockerRegistrySecret.GetName())
	// We must reinitialize to get those values just changed in the default platform configuration
	platform.InitPlatform()

	return nil
}

func getOwnerRef(ctx context.Context, c client.Client) (*metav1.OwnerReference, error) {
	operatorDeploy := &appsv1.Deployment{}

	err := c.Get(ctx, types.NamespacedName{
		// TODO: change theme
		Name:      "camel-k-operator",
		Namespace: "camel-k",
	}, operatorDeploy)
	if err != nil {
		return nil, err
	}

	ownerRef := metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "Deployment",
		Name:               operatorDeploy.Name,
		UID:                operatorDeploy.UID,
		Controller:         new(true),
		BlockOwnerDeletion: new(true),
	}

	return &ownerRef, nil
}

func waitForClusterIP(ctx context.Context, c client.Client, namespace, name string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		var svc corev1.Service

		err := c.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, &svc)
		if err != nil {
			return "", err
		}

		if svc.Spec.ClusterIP != "" &&
			svc.Spec.ClusterIP != corev1.ClusterIPNone {
			return svc.Spec.ClusterIP, nil
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf(
				"timed out waiting for Service %s/%s to get a ClusterIP: %w",
				namespace,
				name,
				ctx.Err(),
			)
		case <-ticker.C:
		}
	}
}
