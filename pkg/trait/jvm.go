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
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"

	infp "gopkg.in/inf.v0"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/envvar"
)

type jvmTrait struct {
	BaseTrait
	traitv1.JVMTrait `property:",squash"`
}

func newJvmTrait() Trait {
	return &jvmTrait{
		BaseTrait: NewBaseTrait("jvm", 2000),
		JVMTrait: traitv1.JVMTrait{
			DebugAddress: "*:5005",
			PrintCommand: pointer.Bool(true),
		},
	}
}

func (t *jvmTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if !e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) || !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if trait := e.Catalog.GetTrait(quarkusTraitID); trait != nil {
		// The JVM trait must be disabled in case the current IntegrationKit corresponds to a native build
		if quarkus, ok := trait.(*quarkusTrait); ok && pointer.BoolDeref(quarkus.Enabled, true) && quarkus.isNativeIntegration(e) {
			return false, nil
		}
	}

	return true, nil
}

//nolint: maintidx // TODO: refactor the code
func (t *jvmTrait) Apply(e *Environment) error {
	kit := e.IntegrationKit

	if kit == nil && e.Integration.Status.IntegrationKit != nil {
		name := e.Integration.Status.IntegrationKit.Name
		ns := e.Integration.GetIntegrationKitNamespace(e.Platform)
		k := v1.NewIntegrationKit(ns, name)
		if err := t.Client.Get(e.Ctx, ctrl.ObjectKeyFromObject(k), k); err != nil {
			return errors.Wrapf(err, "unable to find integration kit %s/%s, %s", ns, name, err)
		}
		kit = k
	}

	if kit == nil {
		if e.Integration.Status.IntegrationKit != nil {
			return fmt.Errorf("unable to find integration kit %s/%s", e.Integration.GetIntegrationKitNamespace(e.Platform), e.Integration.Status.IntegrationKit.Name)
		}
		return fmt.Errorf("unable to find integration kit for integration %s", e.Integration.Name)
	}

	classpath := strset.New()

	classpath.Add("./resources")
	classpath.Add(camel.ConfigResourcesMountPath)
	classpath.Add(camel.ResourcesDefaultMountPath)
	if t.Classpath != "" {
		classpath.Add(strings.Split(t.Classpath, ":")...)
	}

	for _, artifact := range kit.Status.Artifacts {
		classpath.Add(artifact.Target)
	}

	if kit.Labels[v1.IntegrationKitTypeLabel] == v1.IntegrationKitTypeExternal {
		// In case of an external created kit, we do not have any information about
		// the classpath, so we assume the all jars in /deployments/dependencies/ have
		// to be taken into account.
		dependencies := path.Join(builder.DeploymentDir, builder.DependenciesDir)
		classpath.Add(
			dependencies+"/*",
			dependencies+"/app/*",
			dependencies+"/lib/boot/*",
			dependencies+"/lib/main/*",
			dependencies+"/quarkus/*",
		)
	}

	container := e.GetIntegrationContainer()
	if container == nil {
		return fmt.Errorf("unable to find integration container")
	}

	// Build the container command
	// Other traits may have already contributed some arguments
	args := container.Args

	// Remote debugging
	if pointer.BoolDeref(t.Debug, false) {
		suspend := "n"
		if pointer.BoolDeref(t.DebugSuspend, false) {
			suspend = "y"
		}
		args = append(args,
			fmt.Sprintf("-agentlib:jdwp=transport=dt_socket,server=y,suspend=%s,address=%s",
				suspend, t.DebugAddress))

		// Add label to mark the pods with debug enabled
		e.Resources.VisitPodTemplateMeta(func(meta *metav1.ObjectMeta) {
			if meta.Labels == nil {
				meta.Labels = make(map[string]string)
			}
			meta.Labels["camel.apache.org/debug"] = "true"
		})
	}

	hasHeapSizeOption := false
	// Add JVM options
	if len(t.Options) > 0 {
		hasHeapSizeOption = util.StringSliceContainsAnyOf(t.Options, "-Xmx", "-XX:MaxHeapSize", "-XX:MinRAMPercentage", "-XX:MaxRAMPercentage")

		args = append(args, t.Options...)
	}

	// Translate HTTP proxy environment variables, that are set by the environment trait,
	// into corresponding JVM system properties.
	if HTTPProxy := envvar.Get(container.Env, "HTTP_PROXY"); HTTPProxy != nil {
		u, err := url.Parse(HTTPProxy.Value)
		if err != nil {
			return err
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

	if HTTPSProxy := envvar.Get(container.Env, "HTTPS_PROXY"); HTTPSProxy != nil {
		u, err := url.Parse(HTTPSProxy.Value)
		if err != nil {
			return err
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

	// Tune JVM maximum heap size based on the container memory limit, if any.
	// This is configured off-container, thus is limited to explicit user configuration.
	// We may want to inject a wrapper script into the container image, so that it can
	// be performed in-container, based on CGroups memory resource control files.
	if memory, hasLimit := container.Resources.Limits[corev1.ResourceMemory]; !hasHeapSizeOption && hasLimit {
		// Simple heuristic that caps the maximum heap size to 50% of the memory limit
		percentage := int64(50)
		// Unless the memory limit is lower than 300M, in which case we leave more room for the non-heap memory
		if resource.NewScaledQuantity(300, 6).Cmp(memory) > 0 {
			percentage = 25
		}
		memory.AsDec().Mul(memory.AsDec(), infp.NewDec(percentage, 2))
		args = append(args, fmt.Sprintf("-Xmx%dM", memory.ScaledValue(resource.Mega)))
	}

	// Add mounted resources to the class path
	for _, m := range container.VolumeMounts {
		classpath.Add(m.MountPath)
	}
	items := classpath.List()
	// Keep class path sorted so that it's consistent over reconciliation cycles
	sort.Strings(items)
	args = append(args, "-cp", strings.Join(items, ":"))

	args = append(args, e.CamelCatalog.Runtime.ApplicationClass)

	if pointer.BoolDeref(t.PrintCommand, true) {
		args = append([]string{"exec", "java"}, args...)
		container.Command = []string{"/bin/sh", "-c"}
		cmd := strings.Join(args, " ")
		container.Args = []string{"echo " + cmd + " && " + cmd}
	} else {
		container.Command = []string{"java"}
		container.Args = args
	}

	container.WorkingDir = builder.DeploymentDir

	return nil
}

// IsPlatformTrait overrides base class method.
func (t *jvmTrait) IsPlatformTrait() bool {
	return true
}
