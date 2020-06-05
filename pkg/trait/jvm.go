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
	"sort"
	"strings"

	"gopkg.in/inf.v0"

	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// The JVM trait is used to configure the JVM that runs the integration.
//
// +camel-k:trait=jvm
type jvmTrait struct {
	BaseTrait `property:",squash"`
	// Activates remote debugging, so that a debugger can be attached to the JVM, e.g., using port-forwarding
	Debug bool `property:"debug"`
	// Suspends the target JVM immediately before the main class is loaded
	DebugSuspend bool `property:"debug-suspend"`
	// Transport address at which to listen for the newly launched JVM (default `*:5005`)
	DebugAddress string `property:"debug-address"`
	// A comma-separated list of JVM options
	Options *string `property:"options"`
	// Prints the command used the start the JVM in the container logs (default `true`)
	PrintCommand bool `property:"print-command"`
}

func newJvmTrait() Trait {
	return &jvmTrait{
		BaseTrait:    NewBaseTrait("jvm", 2000),
		DebugAddress: "*:5005",
		PrintCommand: true,
	}
}

func (t *jvmTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) ||
		e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseRunning), nil
}

func (t *jvmTrait) Apply(e *Environment) error {
	kit := e.IntegrationKit

	if kit == nil && e.Integration.Status.Kit != "" {
		name := e.Integration.Status.Kit
		k := v1.NewIntegrationKit(e.Integration.Namespace, name)
		key := k8sclient.ObjectKey{
			Namespace: e.Integration.Namespace,
			Name:      name,
		}

		if err := t.Client.Get(t.Ctx, key, &k); err != nil {
			return errors.Wrapf(err, "unable to find integration kit %s, %s", name, err)
		}

		kit = &k
	}

	if kit == nil {
		return fmt.Errorf("unable to find integration kit %s", e.Integration.Status.Kit)
	}

	classpath := strset.New()

	classpath.Add("/etc/camel/resources")
	classpath.Add("./resources")

	for _, artifact := range kit.Status.Artifacts {
		classpath.Add(artifact.Target)
	}

	if kit.Labels["camel.apache.org/kit.type"] == v1.IntegrationKitTypeExternal {
		// In case of an external created kit, we do not have any information about
		// the classpath so we assume the all jars in /deployments/dependencies/ have
		// to be taken into account
		classpath.Add("/deployments/dependencies/*")
	}

	container := e.getIntegrationContainer()
	if container == nil {
		return nil
	}

	// Build the container command
	var args []string

	// Remote debugging
	if t.Debug {
		suspend := "n"
		if t.DebugSuspend {
			suspend = "y"
		}
		args = append(args,
			fmt.Sprintf("-agentlib:jdwp=transport=dt_socket,server=y,suspend=%s,address=%s",
				suspend, t.DebugAddress))
	}

	hasHeapSizeOption := false
	// Add JVM options
	if t.Options != nil {
		hasHeapSizeOption = strings.Contains(*t.Options, "-Xmx") ||
			strings.Contains(*t.Options, "-XX:MaxHeapSize") ||
			strings.Contains(*t.Options, "-XX:MinRAMPercentage") ||
			strings.Contains(*t.Options, "-XX:MaxRAMPercentage")

		args = append(args, strings.Split(*t.Options, ",")...)
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
		memory.AsDec().Mul(memory.AsDec(), inf.NewDec(percentage, 2))
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

	if t.PrintCommand {
		args = append([]string{"java"}, args...)
		container.Command = []string{"/bin/sh", "-c"}
		cmd := strings.Join(args, " ")
		container.Args = []string{"echo " + cmd + " && " + cmd}
	} else {
		container.Command = []string{"java"}
		container.Args = args
	}

	container.WorkingDir = "/deployments"

	return nil
}

// IsPlatformTrait overrides base class method
func (t *jvmTrait) IsPlatformTrait() bool {
	return true
}
