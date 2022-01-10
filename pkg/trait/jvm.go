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

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
)

// The JVM trait is used to configure the JVM that runs the integration.
//
// +camel-k:trait=jvm
type jvmTrait struct {
	BaseTrait `property:",squash"`
	// Activates remote debugging, so that a debugger can be attached to the JVM, e.g., using port-forwarding
	Debug *bool `property:"debug" json:"debug,omitempty"`
	// Suspends the target JVM immediately before the main class is loaded
	DebugSuspend *bool `property:"debug-suspend" json:"debugSuspend,omitempty"`
	// Prints the command used the start the JVM in the container logs (default `true`)
	PrintCommand *bool `property:"print-command" json:"printCommand,omitempty"`
	// Transport address at which to listen for the newly launched JVM (default `*:5005`)
	DebugAddress string `property:"debug-address" json:"debugAddress,omitempty"`
	// A list of JVM options
	Options []string `property:"options" json:"options,omitempty"`
	// Additional JVM classpath (use `Linux` classpath separator)
	Classpath string `property:"classpath" json:"classpath,omitempty"`
}

func newJvmTrait() Trait {
	return &jvmTrait{
		BaseTrait:    NewBaseTrait("jvm", 2000),
		DebugAddress: "*:5005",
		PrintCommand: BoolP(true),
	}
}

func (t *jvmTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	if !e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) || !e.IntegrationInRunningPhases() {
		return false, nil
	}

	return true, nil
}

func (t *jvmTrait) Apply(e *Environment) error {
	kit := e.IntegrationKit

	if kit == nil && e.Integration.Status.IntegrationKit != nil {
		name := e.Integration.Status.IntegrationKit.Name
		ns := e.Integration.GetIntegrationKitNamespace(e.Platform)
		k := v1.NewIntegrationKit(ns, name)
		if err := t.Client.Get(t.Ctx, ctrl.ObjectKeyFromObject(&k), &k); err != nil {
			return errors.Wrapf(err, "unable to find integration kit %s/%s, %s", ns, name, err)
		}
		kit = &k
	}

	if kit == nil {
		if e.Integration.Status.IntegrationKit != nil {
			return fmt.Errorf("unable to find integration kit %s/%s", e.Integration.GetIntegrationKitNamespace(e.Platform), e.Integration.Status.IntegrationKit.Name)
		}
		return fmt.Errorf("unable to find integration kit for integration %s", e.Integration.Name)
	}

	classpath := strset.New()

	classpath.Add("./resources")
	classpath.Add(configResourcesMountPath)
	classpath.Add(resourcesDefaultMountPath)
	if t.Classpath != "" {
		classpath.Add(strings.Split(t.Classpath, ":")...)
	}

	for _, artifact := range kit.Status.Artifacts {
		classpath.Add(artifact.Target)
	}

	if kit.Labels["camel.apache.org/kit.type"] == v1.IntegrationKitTypeExternal {
		// In case of an external created kit, we do not have any information about
		// the classpath so we assume the all jars in /deployments/dependencies/ have
		// to be taken into account
		dependencies := path.Join(builder.DeploymentDir, builder.DependenciesDir)
		classpath.Add(
			dependencies+"/*",
			dependencies+"/app/*",
			dependencies+"/lib/boot/*",
			dependencies+"/lib/main/*",
			dependencies+"/quarkus/*",
		)
	}

	container := e.getIntegrationContainer()
	if container == nil {
		return nil
	}

	// Build the container command
	// Other traits may have already contributed some arguments
	args := container.Args

	// Remote debugging
	if IsTrue(t.Debug) {
		suspend := "n"
		if IsTrue(t.DebugSuspend) {
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
			args = append(args, "-Dhttp.proxyHost="+u.Hostname())
		}
		if !util.StringSliceContainsAnyOf(t.Options, "http.proxyPort") {
			args = append(args, "-Dhttp.proxyPort="+u.Port())
		}
		if user := u.User; !util.StringSliceContainsAnyOf(t.Options, "http.proxyUser") && user != nil {
			args = append(args, "-Dhttp.proxyUser="+user.Username())
			if password, ok := user.Password(); !util.StringSliceContainsAnyOf(t.Options, "http.proxyUser") && ok {
				args = append(args, "-Dhttp.proxyPassword="+password)
			}
		}
	}

	if HTTPSProxy := envvar.Get(container.Env, "HTTPS_PROXY"); HTTPSProxy != nil {
		u, err := url.Parse(HTTPSProxy.Value)
		if err != nil {
			return err
		}
		if !util.StringSliceContainsAnyOf(t.Options, "https.proxyHost") {
			args = append(args, "-Dhttps.proxyHost="+u.Hostname())
		}
		if !util.StringSliceContainsAnyOf(t.Options, "https.proxyPort") {
			args = append(args, "-Dhttps.proxyPort="+u.Port())
		}
		if user := u.User; !util.StringSliceContainsAnyOf(t.Options, "https.proxyUser") && user != nil {
			args = append(args, "-Dhttps.proxyUser="+user.Username())
			if password, ok := user.Password(); !util.StringSliceContainsAnyOf(t.Options, "https.proxyUser") && ok {
				args = append(args, "-Dhttps.proxyPassword="+password)
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
			args = append(args, "-Dhttps.nonProxyHosts="+strings.Join(hosts, "|"))
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

	if IsNilOrTrue(t.PrintCommand) {
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

// IsPlatformTrait overrides base class method
func (t *jvmTrait) IsPlatformTrait() bool {
	return true
}
