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

package openshift

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IsOpenShift returns true if we are connected to a OpenShift cluster.
func IsOpenShift(client kubernetes.Interface) (bool, error) {
	_, err := client.Discovery().ServerResourcesForGroupVersion("image.openshift.io/v1")
	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// GetOpenshiftPodSecurityContextRestricted return the PodSecurityContext (https://docs.openshift.com/container-platform/4.12/authentication/managing-security-context-constraints.html):
// FsGroup set to the minimum value in the "openshift.io/sa.scc.supplemental-groups" annotation if exists, else falls back to minimum value "openshift.io/sa.scc.uid-range" annotation.
func GetOpenshiftPodSecurityContextRestricted(ctx context.Context, client kubernetes.Interface, namespace string) (*corev1.PodSecurityContext, error) {

	ns, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %q: %w", namespace, err)
	}

	uidRange, ok := ns.ObjectMeta.Annotations["openshift.io/sa.scc.uid-range"]
	if !ok {
		return nil, errors.New("annotation 'openshift.io/sa.scc.uid-range' not found")
	}

	supplementalGroups, ok := ns.ObjectMeta.Annotations["openshift.io/sa.scc.supplemental-groups"]
	if !ok {
		supplementalGroups = uidRange
	}

	supplementalGroups = strings.Split(supplementalGroups, ",")[0]
	fsGroupStr := strings.Split(supplementalGroups, "/")[0]
	fsGroup, err := strconv.ParseInt(fsGroupStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert fsgroup to integer %q: %w", fsGroupStr, err)
	}

	psc := corev1.PodSecurityContext{
		FSGroup: &fsGroup,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	return &psc, nil

}

// GetOpenshiftSecurityContextRestricted return the PodSecurityContext (https://docs.openshift.com/container-platform/4.12/authentication/managing-security-context-constraints.html):
// User set to the minimum value in the "openshift.io/sa.scc.uid-range" annotation.
func GetOpenshiftSecurityContextRestricted(ctx context.Context, client kubernetes.Interface, namespace string) (*corev1.SecurityContext, error) {

	ns, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %q: %w", namespace, err)
	}

	uidRange, ok := ns.ObjectMeta.Annotations["openshift.io/sa.scc.uid-range"]
	if !ok {
		return nil, errors.New("annotation 'openshift.io/sa.scc.uid-range' not found")
	}

	uidStr := strings.Split(uidRange, "/")[0]
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert uid to integer %q: %w", uidStr, err)
	}

	runAsNonRoot := true
	allowPrivilegeEscalation := false
	sc := corev1.SecurityContext{
		RunAsUser:    &uid,
		RunAsNonRoot: &runAsNonRoot,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}

	return &sc, nil

}

// GetOpenshiftUser return the UserId (https://docs.openshift.com/container-platform/4.12/authentication/managing-security-context-constraints.html):
// User set to the minimum value in the "openshift.io/sa.scc.uid-range" annotation.
func GetOpenshiftUser(ctx context.Context, client kubernetes.Interface, namespace string) (string, error) {

	ns, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get namespace %q: %w", namespace, err)
	}

	uidRange, ok := ns.ObjectMeta.Annotations["openshift.io/sa.scc.uid-range"]
	if !ok {
		return "", errors.New("annotation 'openshift.io/sa.scc.uid-range' not found")
	}

	uidStr := strings.Split(uidRange, "/")[0]
	return uidStr, nil
}
