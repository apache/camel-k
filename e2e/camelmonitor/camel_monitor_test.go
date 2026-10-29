//go:build integration
// +build integration

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

package camelmonitor

import (
	"context"
	"testing"

	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
)

func TestCamelMonitorDiscovery(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		const discoveryLabel = "camel.apache.org/monitor"
		InstallOperator(t, ctx, g, ns)

		name := "monitored-integration"
		g.Expect(KamelRun(t, ctx, ns, "files/timer.yaml", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
			Should(Equal(corev1.ConditionTrue))
		g.Expect(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionCamelMonitorAvailable)()).To(BeNil())

		t.Run("direct label", func(t *testing.T) {
			monitor := monitorv1alpha1.NewCamelMonitor(ns, "direct-monitor")
			monitor.Labels = map[string]string{discoveryLabel: name}
			assertMonitorLifecycle(t, ctx, ns, name, &monitor, nil)
		})

		// Auxiliary sources have different names from the Integration. They do not
		// replace or modify its running workload; only their metadata is needed.
		image := IntegrationStatusImage(t, ctx, ns, name)()
		g.Expect(image).NotTo(BeEmpty())
		for _, source := range monitorSources(ns, name, discoveryLabel, image) {
			t.Run(source.kind, func(t *testing.T) {
				g := NewWithT(t)
				g.Expect(TestClient(t).Create(ctx, source.object)).To(Succeed())
				monitor := monitorv1alpha1.NewCamelMonitor(ns, "monitor-"+source.object.GetName())
				monitor.Annotations = map[string]string{
					monitorv1alpha1.MonitorImportedKindLabel: source.kind,
					monitorv1alpha1.MonitorImportedNameLabel: source.object.GetName(),
				}
				assertMonitorLifecycle(t, ctx, ns, name, &monitor, nil)
			})
		}
		for _, sourceChange := range []string{"remove-label", "delete-source", "change-integration"} {
			t.Run(sourceChange, func(t *testing.T) {
				g := NewWithT(t)
				c := TestClient(t)
				source := monitorSources(ns, name, discoveryLabel, image)[1].object
				sourceName := sourceChange + "-source"
				source.SetName(sourceName)
				g.Expect(c.Create(ctx, source)).To(Succeed())
				monitor := monitorv1alpha1.NewCamelMonitor(ns, "monitor-"+sourceName)
				monitor.Annotations = map[string]string{
					monitorv1alpha1.MonitorImportedKindLabel: "CronJob",
					monitorv1alpha1.MonitorImportedNameLabel: source.GetName(),
				}
				assertMonitorLifecycle(t, ctx, ns, name, &monitor, func() {
					switch sourceChange {
					case "delete-source":
						g.Expect(c.Delete(ctx, source)).To(Succeed())
					case "remove-label":
						labels := source.GetLabels()
						delete(labels, discoveryLabel)
						source.SetLabels(labels)
						g.Expect(c.Update(ctx, source)).To(Succeed())
					case "change-integration":
						labels := source.GetLabels()
						labels[v1.IntegrationLabel] = "other-integration"
						source.SetLabels(labels)
						g.Expect(c.Update(ctx, source)).To(Succeed())
					}
				})
				if sourceChange != "delete-source" {
					g.Expect(c.Delete(ctx, source)).To(Succeed())
				}
			})
		}
	})
}

func assertMonitorLifecycle(t *testing.T, ctx context.Context, ns, integration string, monitor *monitorv1alpha1.CamelMonitor, beforeDelete func()) {
	t.Helper()
	g := NewWithT(t)
	c := TestClient(t)
	g.Expect(c.Create(ctx, monitor)).To(Succeed())
	t.Cleanup(func() {
		if err := c.Delete(ctx, monitor); err != nil && !k8serrors.IsNotFound(err) {
			t.Errorf("cannot clean up CamelMonitor %s: %v", monitor.Name, err)
		}
	})
	g.Eventually(IntegrationConditionStatus(t, ctx, ns, integration, v1.IntegrationConditionCamelMonitorAvailable), TestTimeoutShort).
		Should(Equal(corev1.ConditionTrue))

	if beforeDelete != nil {
		beforeDelete()
	}
	// The monitor's deletion must clear availability even if its source lost discovery.
	g.Expect(c.Delete(ctx, monitor)).To(Succeed())
	g.Eventually(IntegrationCondition(t, ctx, ns, integration, v1.IntegrationConditionCamelMonitorAvailable), TestTimeoutShort).
		Should(BeNil())
	g.Expect(IntegrationPhase(t, ctx, ns, integration)()).To(Equal(v1.IntegrationPhaseRunning))
	g.Expect(IntegrationConditionStatus(t, ctx, ns, integration, v1.IntegrationConditionReady)()).To(Equal(corev1.ConditionTrue))
}

func monitorSources(ns, integration, discoveryLabel, image string) []struct {
	kind   string
	object ctrl.Object
} {
	metadata := func(name string) metav1.ObjectMeta {
		return metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels: map[string]string{
				discoveryLabel:      "discovered-source",
				v1.IntegrationLabel: integration,
			},
		}
	}
	podSpec := corev1.PodSpec{Containers: []corev1.Container{{Name: "integration", Image: image}}}
	cronPodSpec := *podSpec.DeepCopy()
	cronPodSpec.RestartPolicy = corev1.RestartPolicyNever
	return []struct {
		kind   string
		object ctrl.Object
	}{
		{
			kind: "Deployment",
			object: &appsv1.Deployment{
				ObjectMeta: metadata("deployment-source"),
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(0)),
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "deployment-source"}},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "deployment-source"}},
						Spec:       podSpec,
					},
				},
			},
		},
		{
			kind: "CronJob",
			object: &batchv1.CronJob{
				ObjectMeta: metadata("cronjob-source"),
				Spec: batchv1.CronJobSpec{
					Schedule: "0 0 * * *",
					Suspend:  ptr.To(true),
					JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{Spec: cronPodSpec},
					}},
				},
			},
		},
		{
			kind: "Service",
			object: &servingv1.Service{
				ObjectMeta: metadata("knative-source"),
				Spec: servingv1.ServiceSpec{ConfigurationSpec: servingv1.ConfigurationSpec{
					Template: servingv1.RevisionTemplateSpec{
						Spec: servingv1.RevisionSpec{PodSpec: podSpec},
					},
				}},
			},
		},
	}
}
