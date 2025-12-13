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

package trait

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/sets"
)

const (
	jvmTraitID    = "jvm"
	jvmTraitOrder = 2000

	defaultMaxMemoryScale               = 6
	defaultMaxMemoryPercentage          = int64(50)
	lowMemoryThreshold                  = 300
	lowMemoryMAxMemoryDefaultPercentage = int64(25)
	defaultCACertMountPath              = "/etc/camel/conf.d/_truststore"
	caCertVolumeName                    = "jvm-truststore"
	caCertSecretVolumeName              = "ca-cert-secret" //nolint:gosec // G101: not a credential, just a volume name
	trustStoreName                      = "truststore.jks"
)

type jvmTrait struct {
	BaseTrait
	traitv1.JVMTrait `property:",squash"`

	jvmAgentArgs []string
}

func newJvmTrait() Trait {
	return &jvmTrait{
		BaseTrait: NewBaseTrait(jvmTraitID, jvmTraitOrder),
	}
}

func (t *jvmTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	// Deprecated: the JVM has to be a platform trait and the user should not be able to disable it
	if !ptr.Deref(t.Enabled, true) {
		notice := userDisabledMessage + "; this configuration is deprecated and may be removed within next releases"

		return false, NewIntegrationCondition("JVM", v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, TraitConfigurationReason, notice), nil
	}

	if (e.IntegrationKit != nil && !e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady)) || !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	//nolint: staticcheck
	if ((e.Integration != nil && !e.Integration.IsManagedBuild()) || (e.IntegrationKit != nil && e.IntegrationKit.IsSynthetic())) &&
		t.Jar == "" {
		// We skip this trait since we cannot make any assumption on the container Java tooling running
		// for the synthetic IntegrationKit
		return false, NewIntegrationConditionPlatformDisabledWithMessage(
			"JVM",
			"integration kit was not created via Camel K operator and the user did not provide the jar to execute",
		), nil
	}

	// The JVM trait must be disabled in case the current IntegrationKit corresponds to a native build
	if qt := e.Catalog.GetTrait(quarkusTraitID); qt != nil {
		if quarkus, ok := qt.(*quarkusTrait); ok && quarkus.isNativeIntegration(e) {
			return false, NewIntegrationConditionPlatformDisabledWithMessage("JVM", "quarkus native build"), nil
		}
	}

	if t.Jar == "" && e.Integration.Status.Jar != "" {
		t.Jar = e.Integration.Status.Jar
	}

	if t.Agents != nil {
		jvmAgentArgs, err := t.parseJvmAgentsArgs()
		if err != nil {
			return false, nil, err
		}
		t.jvmAgentArgs = jvmAgentArgs
	}

	return true, nil, nil
}

func (t *jvmTrait) Apply(e *Environment) error {
	container := e.GetIntegrationContainer()
	if container == nil {
		return fmt.Errorf("unable to find a container for %s Integration", e.Integration.Name)
	}

	// Build the container command
	// Other traits may have already contributed some arguments
	args := container.Args

	if ptr.Deref(t.Debug, false) {
		debugArgs := t.enableDebug(e)
		args = append(args, debugArgs)
	}

	// JVM agents: they should have been parsed during configuration phase
	if t.jvmAgentArgs != nil {
		args = append(args, t.jvmAgentArgs...)
	}

	hasHeapSizeOption := false
	// Add JVM options
	if len(t.Options) > 0 {
		hasHeapSizeOption = util.StringSliceContainsAnyOf(t.Options, "-Xmx", "-XX:MaxHeapSize", "-XX:MinRAMPercentage", "-XX:MaxRAMPercentage")
		args = append(args, t.Options...)
	}

	// Tune JVM maximum heap size based on the container memory limit, if any.
	// This is configured off-container, thus is limited to explicit user configuration.
	// We may want to inject a wrapper script into the container image, so that it can
	// be performed in-container, based on CGroups memory resource control files.
	if memory, hasLimit := container.Resources.Limits[corev1.ResourceMemory]; !hasHeapSizeOption && hasLimit {
		// Simple heuristic that caps the maximum heap size to 50% of the memory limit
		percentage := defaultMaxMemoryPercentage
		// Unless the memory limit is lower than 300M, in which case we leave more room for the non-heap memory
		if resource.NewScaledQuantity(lowMemoryThreshold, defaultMaxMemoryScale).Cmp(memory) > 0 {
			percentage = lowMemoryMAxMemoryDefaultPercentage
		}
		//nolint:mnd
		memScaled := memory.ScaledValue(resource.Mega) * percentage / 100
		args = append(args, fmt.Sprintf("-Xmx%dM", memScaled))
	}

	httpProxyArgs, err := t.prepareHTTPProxy(container)
	if err != nil {
		return err
	}
	if httpProxyArgs != nil {
		args = append(args, httpProxyArgs...)
	}

	caCertArgs, err := t.configureCaCert(e)
	if err != nil {
		return err
	}
	if caCertArgs != nil {
		args = append(args, caCertArgs...)
	}

	return t.feedContainer(container, args, e)
}

//nolint:nestif
func (t *jvmTrait) feedContainer(container *corev1.Container, args []string, e *Environment) error {
	// If user provided the jar, we will execute on the container something like
	// java -Dxyx ... -cp ... -jar my-app.jar
	// For this reason it's important that the container is a java based container able to run a Camel (hence Java) application
	container.WorkingDir = builder.DeploymentDir
	container.Command = []string{"java"}
	classpathItems := t.prepareClasspathItems(container)
	if t.Jar != "" {
		// User is providing the Jar to execute explicitly
		args = append(args, "-cp", classpathItems)
		args = append(args, "-jar", t.Jar)
	} else {
		kit, err := t.getIntegrationKit(e)
		if err != nil {
			return err
		}
		if kit != nil {
			// managed Integrations
			kitDepsDirs := kit.Status.GetDependenciesPaths()
			if kitDepsDirs.IsEmpty() {
				// Use legacy Camel Quarkus expected structure
				kitDepsDirs = getLegacyCamelQuarkusDependenciesPaths()
			}
			classpathItems = getClasspath(kitDepsDirs, classpathItems)
		}
		args = append(args, "-cp", classpathItems)
		args = append(args, e.CamelCatalog.Runtime.ApplicationClass)
	}
	container.Args = args

	return nil
}

// getClasspath merges the classpath required by the kit with any value provided in the trait.
func getClasspath(depsDirs *sets.Set, jvmTraitClasspath string) string {
	if !depsDirs.IsEmpty() {
		if jvmTraitClasspath != "" {
			jvmTraitClasspathSet := getClasspathSet(jvmTraitClasspath)
			depsDirs = sets.Union(depsDirs, jvmTraitClasspathSet)
		}
		classPaths := depsDirs.List()
		sort.Strings(classPaths)

		return strings.Join(classPaths, ":")
	}

	return jvmTraitClasspath
}

func getClasspathSet(cps string) *sets.Set {
	s := sets.NewSet()
	for _, cp := range strings.Split(cps, ":") {
		s.Add(cp)
	}

	return s
}

func (t *jvmTrait) getIntegrationKit(e *Environment) (*v1.IntegrationKit, error) {
	kit := e.IntegrationKit

	if kit == nil && e.Integration.Status.IntegrationKit != nil {
		name := e.Integration.Status.IntegrationKit.Name
		ns := e.Integration.GetIntegrationKitNamespace(e.Platform)
		k, err := kubernetes.GetIntegrationKit(e.Ctx, t.Client, name, ns)
		if err != nil {
			return nil, fmt.Errorf("unable to find integration kit %s/%s: %w", ns, name, err)
		}
		kit = k
	}

	return kit, nil
}

func (t *jvmTrait) enableDebug(e *Environment) string {
	suspend := "n"
	if ptr.Deref(t.DebugSuspend, false) {
		suspend = "y"
	}
	// Add label to mark the pods with debug enabled
	e.Resources.VisitPodTemplateMeta(func(meta *metav1.ObjectMeta) {
		if meta.Labels == nil {
			meta.Labels = make(map[string]string)
		}
		meta.Labels["camel.apache.org/debug"] = trueString
	})
	t.DebugAddress = "*:5005"

	return fmt.Sprintf("-agentlib:jdwp=transport=dt_socket,server=y,suspend=%s,address=%s",
		suspend, t.DebugAddress)
}

func (t *jvmTrait) prepareClasspathItems(container *corev1.Container) string {
	existingClasspaths := extractExistingClasspathItems(container)
	classpath := sets.NewSet()
	// Deprecated: replaced by /etc/camel/resources.d/[_configmaps/_secrets] (camel.ResourcesConfigmapsMountPath/camel.ResourcesSecretsMountPath).
	classpath.Add("./resources")
	classpath.Add(filepath.ToSlash(camel.ResourcesConfigmapsMountPath))
	classpath.Add(filepath.ToSlash(camel.ResourcesSecretsMountPath))
	// Deprecated: replaced by /etc/camel/resources.d/[_configmaps/_secrets] (camel.ResourcesConfigmapsMountPath/camel.ResourcesSecretsMountPath).
	//nolint: staticcheck
	classpath.Add(filepath.ToSlash(camel.ResourcesDefaultMountPath))
	if t.Classpath != "" {
		classpath.Add(strings.Split(t.Classpath, ":")...)
	}
	// Add mounted resources to the class path
	for _, m := range container.VolumeMounts {
		classpath.Add(m.MountPath)
	}
	items := classpath.List()
	// Keep class path sorted so that it's consistent over reconciliation cycles
	sort.Strings(items)

	if existingClasspaths != nil {
		existingClasspaths = append(existingClasspaths, items...)

		return strings.Join(existingClasspaths, ":")
	}

	return strings.Join(items, ":")
}

// extractExistingClasspathItems returns any container classpath option (if exists).
func extractExistingClasspathItems(container *corev1.Container) []string {
	for i, arg := range container.Args {
		if arg == "-cp" || arg == "-classpath" {
			if i < len(container.Args) {
				// return the next argument
				return strings.Split(container.Args[i+1], ":")
			}
		}
	}

	return nil
}

// Translate HTTP proxy environment variables, that are set by the environment trait,
// into corresponding JVM system properties.
func (t *jvmTrait) prepareHTTPProxy(container *corev1.Container) ([]string, error) {
	var args []string

	//nolint:dupl,nestif
	if HTTPProxy := envvar.Get(container.Env, "HTTP_PROXY"); HTTPProxy != nil {
		u, err := url.Parse(HTTPProxy.Value)
		if err != nil {
			return args, err
		}
		if !util.StringSliceContainsAnyOf(t.Options, "http.proxyHost") {
			args = append(args, fmt.Sprintf("-Dhttp.proxyHost=%q", u.Hostname()))
		}
		if port := u.Port(); !util.StringSliceContainsAnyOf(t.Options, "http.proxyPort") && port != "" {
			args = append(args, fmt.Sprintf("-Dhttp.proxyPort=%q", u.Port()))
		}
		if user := u.User; !util.StringSliceContainsAnyOf(t.Options, "http.proxyUser") && user != nil {
			args = append(args, fmt.Sprintf("-Dhttp.proxyUser=%q", user.Username()))
			if password, ok := user.Password(); !util.StringSliceContainsAnyOf(t.Options, "http.proxyUser") && ok {
				args = append(args, fmt.Sprintf("-Dhttp.proxyPassword=%q", password))
			}
		}
	}

	//nolint:dupl,nestif
	if HTTPSProxy := envvar.Get(container.Env, "HTTPS_PROXY"); HTTPSProxy != nil {
		u, err := url.Parse(HTTPSProxy.Value)
		if err != nil {
			return args, err
		}
		if !util.StringSliceContainsAnyOf(t.Options, "https.proxyHost") {
			args = append(args, fmt.Sprintf("-Dhttps.proxyHost=%q", u.Hostname()))
		}
		if port := u.Port(); !util.StringSliceContainsAnyOf(t.Options, "https.proxyPort") && port != "" {
			args = append(args, fmt.Sprintf("-Dhttps.proxyPort=%q", u.Port()))
		}
		if user := u.User; !util.StringSliceContainsAnyOf(t.Options, "https.proxyUser") && user != nil {
			args = append(args, fmt.Sprintf("-Dhttps.proxyUser=%q", user.Username()))
			if password, ok := user.Password(); !util.StringSliceContainsAnyOf(t.Options, "https.proxyUser") && ok {
				args = append(args, fmt.Sprintf("-Dhttps.proxyPassword=%q", password))
			}
		}
	}

	if noProxy := envvar.Get(container.Env, "NO_PROXY"); noProxy != nil {
		if !util.StringSliceContainsAnyOf(t.Options, "http.nonProxyHosts") {
			// Convert to the format expected by the JVM http.nonProxyHosts system property
			hosts := strings.Split(strings.ReplaceAll(noProxy.Value, " ", ""), ",")
			for i, host := range hosts {
				if strings.HasPrefix(host, ".") {
					hosts[i] = strings.Replace(host, ".", "*.", 1)
				}
			}
			args = append(args, fmt.Sprintf("-Dhttp.nonProxyHosts=%q", strings.Join(hosts, "|")))
		}
	}

	return args, nil
}

// Deprecated: to be removed as soon as version 2.3.x is no longer supported.
func getLegacyCamelQuarkusDependenciesPaths() *sets.Set {
	s := sets.NewSet()
	s.Add("dependencies/*")
	s.Add("dependencies/lib/boot/*")
	s.Add("dependencies/lib/main/*")
	s.Add("dependencies/quarkus/*")

	return s
}

// parseSecretRef parses a secret reference in the format "secret:name" or "secret:name/key".
func parseSecretRef(ref string) (string, string, error) {
	if !strings.HasPrefix(ref, "secret:") {
		return "", "", fmt.Errorf("invalid CA cert reference %q: must start with 'secret:'", ref)
	}

	ref = strings.TrimPrefix(ref, "secret:")
	parts := strings.SplitN(ref, "/", 2)
	secretName, secretKey := parts[0], ""

	if len(parts) > 1 {
		secretKey = parts[1]
	}
	if secretName == "" {
		return "", "", errors.New("invalid CA cert reference: secret name is empty")
	}

	return secretName, secretKey, nil
}

// configureCACert sets up the truststore for CA certificates.
func (t *jvmTrait) configureCaCert(e *Environment) ([]string, error) {
	if t.CACert == "" {
		return nil, nil
	}

	secretName, secretKey, err := parseSecretRef(t.CACert)
	if err != nil {
		return nil, err
	}

	if secretKey == "" {
		secretKey = "ca.crt"
	}

	mountPath := defaultCACertMountPath
	if t.CACertMountPath != "" {
		mountPath = t.CACertMountPath
	}

	// Use a deterministic password based on integration name to avoid
	// changing the deployment spec on every reconciliation cycle.
	// For a truststore i.e public CA certs only, security of this password is not critical.
	trustStorePass := "camelk-" + e.Integration.Name
	trustStorePath := filepath.Join(mountPath, trustStoreName)

	// add secret volume.
	secretVolume := corev1.Volume{
		Name: caCertSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}

	// add an emptyDir volume.
	trustStoreVolume := corev1.Volume{
		Name: caCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	// add volumes to deployment.
	e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		deployment.Spec.Template.Spec.Volumes = append(
			deployment.Spec.Template.Spec.Volumes,
			secretVolume, trustStoreVolume,
		)
	})

	// add mount to integration container
	container := e.GetIntegrationContainer()
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      caCertVolumeName,
		MountPath: mountPath,
		ReadOnly:  true,
	})

	initContainer := corev1.Container{
		Name:            "generate-truststore",
		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
		Command: []string{
			"keytool",
			"-importcert",
			"-noprompt",
			"-alias",
			"custom-ca",
			"-storepass",
			trustStorePass,
			"-keystore",
			trustStorePath,
			"-file",
			filepath.Join("/etc/secrets/cacert", secretKey),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      caCertSecretVolumeName,
				MountPath: "/etc/secrets/cacert",
				ReadOnly:  true,
			},
			{
				Name:      caCertVolumeName,
				MountPath: mountPath,
			},
		},
	}

	// add to deployment container
	e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		deployment.Spec.Template.Spec.InitContainers = append(
			deployment.Spec.Template.Spec.InitContainers,
			initContainer,
		)
	})

	return []string{
		"-Djavax.net.ssl.trustStore=" + trustStorePath,
		"-Djavax.net.ssl.trustStorePassword=" + trustStorePass,
	}, nil
}
