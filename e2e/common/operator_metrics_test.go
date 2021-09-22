// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package common

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"

	prometheus "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/apache/camel-k/e2e/support/util"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestMetrics(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "java"
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"-t", "prometheus.enabled=true",
			"-t", "prometheus.pod-monitor=false").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		pod := OperatorPod(ns)()
		Expect(pod).NotTo(BeNil())

		logs := StructuredLogs(ns, pod.Name, corev1.PodLogOptions{}, false)
		Expect(logs).NotTo(BeEmpty())

		response, err := TestClient().CoreV1().RESTClient().Get().
			AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/metrics", ns, pod.Name)).DoRaw(TestContext)
		Expect(err).To(BeNil())
		metrics, err := parsePrometheusData(response)
		Expect(err).To(BeNil())

		it := Integration(ns, name)()
		Expect(it).NotTo(BeNil())
		build := Build(ns, it.Status.IntegrationKit.Name)()
		Expect(build).NotTo(BeNil())

		t.Run("Build duration metric", func(t *testing.T) {
			// Get the duration from the Build status
			duration, err := time.ParseDuration(build.Status.Duration)
			Expect(err).To(BeNil())

			// Check it's consistent with the duration observed from logs
			var ts1, ts2 time.Time
			err = NewLogWalker(&logs).
				AddStep(MatchFields(IgnoreExtras, Fields{
					"LoggerName":  Equal("camel-k.controller.build"),
					"Message":     Equal("state transition"),
					"PhaseFrom":   Equal(string(v1.BuildPhaseScheduling)),
					"PhaseTo":     Equal(string(v1.BuildPhasePending)),
					"RequestName": Equal(build.Name),
				}), func(l *LogEntry) { ts1 = l.Timestamp.Time }).
				AddStep(MatchFields(IgnoreExtras, Fields{
					"LoggerName": Equal("camel-k.builder"),
					"Message":    HavePrefix("resolved base image:"),
				}), LogEntryNoop).
				AddStep(MatchFields(IgnoreExtras, Fields{
					"LoggerName":  Equal("camel-k.controller.build"),
					"Message":     Equal("state transition"),
					"PhaseFrom":   Equal(string(v1.BuildPhaseRunning)),
					"PhaseTo":     Equal(string(v1.BuildPhaseSucceeded)),
					"RequestName": Equal(build.Name),
				}), func(l *LogEntry) { ts2 = l.Timestamp.Time }).
				Walk()
			Expect(err).To(BeNil())
			Expect(ts1).NotTo(BeZero())
			Expect(ts2).NotTo(BeZero())
			Expect(ts2).To(BeTemporally(">", ts1))

			durationFromLogs := ts2.Sub(ts1)
			Expect(math.Abs((durationFromLogs - duration).Seconds())).To(BeNumerically("<", 1))

			// Check the duration is observed in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_build_duration_seconds",
				EqualP(prometheus.MetricFamily{
					Name: stringP("camel_k_build_duration_seconds"),
					Help: stringP("Camel K build duration"),
					Type: metricTypeP(prometheus.MetricType_HISTOGRAM),
					Metric: []*prometheus.Metric{
						{
							Label: []*prometheus.LabelPair{
								label("result", "Succeeded"),
								label("type", "fast-jar"),
							},
							Histogram: &prometheus.Histogram{
								SampleCount: uint64P(1),
								SampleSum:   float64P(duration.Seconds()),
								Bucket:      buckets(duration.Seconds(), []float64{30, 60, 90, 120, 300, 600, math.Inf(1)}),
							},
						},
					},
				}),
			))
		})

		t.Run("Build recovery attempts metric", func(t *testing.T) {
			// Check there are no failures reported in the Build status
			Expect(build.Status.Failure).To(BeNil())

			// Check no recovery attempts are reported in the logs
			recoveryAttempts, err := NewLogCounter(&logs).Count(MatchFields(IgnoreExtras, Fields{
				"LoggerName":  Equal("camel-k.controller.build"),
				"Message":     HavePrefix("Recovery attempt"),
				"Kind":        Equal("Build"),
				"RequestName": Equal(build.Name),
			}))
			Expect(err).To(BeNil())
			Expect(recoveryAttempts).To(BeNumerically("==", 0))

			// Check no recovery attempts are observed in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_build_recovery_attempts",
				EqualP(prometheus.MetricFamily{
					Name: stringP("camel_k_build_recovery_attempts"),
					Help: stringP("Camel K build recovery attempts"),
					Type: metricTypeP(prometheus.MetricType_HISTOGRAM),
					Metric: []*prometheus.Metric{
						{
							Label: []*prometheus.LabelPair{
								label("result", "Succeeded"),
								label("type", "fast-jar"),
							},
							Histogram: &prometheus.Histogram{
								SampleCount: uint64P(1),
								SampleSum:   float64P(0),
								Bucket:      buckets(0, []float64{0, 1, 2, 3, 4, 5, math.Inf(1)}),
							},
						},
					},
				}),
			))
		})

		t.Run("reconciliation duration metric", func(t *testing.T) {
			Expect(metrics).To(HaveKeyWithValue("camel_k_reconciliation_duration_seconds",
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Name": EqualP("camel_k_reconciliation_duration_seconds"),
					"Help": EqualP("Camel K reconciliation loop duration"),
					"Type": EqualP(prometheus.MetricType_HISTOGRAM),
				})),
			))

			counter := NewLogCounter(&logs)

			// Count the number of IntegrationPlatform reconciliations
			platformReconciliations, err := counter.Count(MatchFields(IgnoreExtras, Fields{
				"LoggerName":       Equal("camel-k.controller.integrationplatform"),
				"Message":          Equal("Reconciling IntegrationPlatform"),
				"RequestNamespace": Equal(ns),
				"RequestName":      Equal("camel-k"),
			}))
			Expect(err).To(BeNil())

			// Check it matches the observation in the corresponding metric
			platformReconciled := getMetric(metrics["camel_k_reconciliation_duration_seconds"],
				MatchFieldsP(IgnoreExtras, Fields{
					"Label": ConsistOf(
						label("group", v1.SchemeGroupVersion.Group),
						label("version", v1.SchemeGroupVersion.Version),
						label("kind", "IntegrationPlatform"),
						label("namespace", ns),
						label("result", "Reconciled"),
						label("tag", ""),
					),
				}))
			Expect(platformReconciled).NotTo(BeNil())
			platformReconciledCount := *platformReconciled.Histogram.SampleCount
			Expect(platformReconciledCount).To(BeNumerically(">", 0))

			platformRequeued := getMetric(metrics["camel_k_reconciliation_duration_seconds"],
				MatchFieldsP(IgnoreExtras, Fields{
					"Label": ConsistOf(
						label("group", v1.SchemeGroupVersion.Group),
						label("version", v1.SchemeGroupVersion.Version),
						label("kind", "IntegrationPlatform"),
						label("namespace", ns),
						label("result", "Requeued"),
						label("tag", ""),
					),
				}))
			platformRequeuedCount := uint64(0)
			if platformRequeued != nil {
				platformRequeuedCount = *platformRequeued.Histogram.SampleCount
			}

			Expect(platformReconciliations).To(BeNumerically("==", platformReconciledCount+platformRequeuedCount))

			// Count the number of Integration reconciliations
			integrationReconciliations, err := counter.Count(MatchFields(IgnoreExtras, Fields{
				"LoggerName":       Equal("camel-k.controller.integration"),
				"Message":          Equal("Reconciling Integration"),
				"RequestNamespace": Equal(it.Namespace),
				"RequestName":      Equal(it.Name),
			}))
			Expect(err).To(BeNil())
			Expect(integrationReconciliations).To(BeNumerically(">", 0))

			// Check it matches the observation in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_reconciliation_duration_seconds",
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Name": EqualP("camel_k_reconciliation_duration_seconds"),
					"Help": EqualP("Camel K reconciliation loop duration"),
					"Type": EqualP(prometheus.MetricType_HISTOGRAM),
					"Metric": ContainElement(MatchFieldsP(IgnoreExtras, Fields{
						"Label": ConsistOf(
							label("group", v1.SchemeGroupVersion.Group),
							label("version", v1.SchemeGroupVersion.Version),
							label("kind", "Integration"),
							label("namespace", it.Namespace),
							label("result", "Reconciled"),
							label("tag", ""),
						),
						"Histogram": MatchFieldsP(IgnoreExtras, Fields{
							"SampleCount": EqualP(uint64(integrationReconciliations)),
						}),
					})),
				})),
			))

			// Count the number of IntegrationKit reconciliations
			integrationKitReconciliations, err := counter.Count(MatchFields(IgnoreExtras, Fields{
				"LoggerName":       Equal("camel-k.controller.integrationkit"),
				"Message":          Equal("Reconciling IntegrationKit"),
				"RequestNamespace": Equal(it.Status.IntegrationKit.Namespace),
				"RequestName":      Equal(it.Status.IntegrationKit.Name),
			}))
			Expect(err).To(BeNil())
			Expect(integrationKitReconciliations).To(BeNumerically(">", 0))

			// Check it matches the observation in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_reconciliation_duration_seconds",
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Name": EqualP("camel_k_reconciliation_duration_seconds"),
					"Help": EqualP("Camel K reconciliation loop duration"),
					"Type": EqualP(prometheus.MetricType_HISTOGRAM),
					"Metric": ContainElement(MatchFieldsP(IgnoreExtras, Fields{
						"Label": ConsistOf(
							label("group", v1.SchemeGroupVersion.Group),
							label("version", v1.SchemeGroupVersion.Version),
							label("kind", "IntegrationKit"),
							label("namespace", it.Status.IntegrationKit.Namespace),
							label("result", "Reconciled"),
							label("tag", ""),
						),
						"Histogram": MatchFieldsP(IgnoreExtras, Fields{
							"SampleCount": EqualP(uint64(integrationKitReconciliations)),
						}),
					})),
				})),
			))

			// Count the number of Build reconciliations
			buildReconciliations, err := counter.Count(MatchFields(IgnoreExtras, Fields{
				"LoggerName":       Equal("camel-k.controller.build"),
				"Message":          Equal("Reconciling Build"),
				"RequestNamespace": Equal(build.Namespace),
				"RequestName":      Equal(build.Name),
			}))
			Expect(err).To(BeNil())

			// Check it matches the observation in the corresponding metric
			buildReconciled := getMetric(metrics["camel_k_reconciliation_duration_seconds"],
				MatchFieldsP(IgnoreExtras, Fields{
					"Label": ConsistOf(
						label("group", v1.SchemeGroupVersion.Group),
						label("version", v1.SchemeGroupVersion.Version),
						label("kind", "Build"),
						label("namespace", build.Namespace),
						label("result", "Reconciled"),
						label("tag", ""),
					),
				}))
			Expect(buildReconciled).NotTo(BeNil())
			buildReconciledCount := *buildReconciled.Histogram.SampleCount
			Expect(buildReconciledCount).To(BeNumerically(">", 0))

			buildRequeued := getMetric(metrics["camel_k_reconciliation_duration_seconds"],
				MatchFieldsP(IgnoreExtras, Fields{
					"Label": ConsistOf(
						label("group", v1.SchemeGroupVersion.Group),
						label("version", v1.SchemeGroupVersion.Version),
						label("kind", "Build"),
						label("namespace", build.Namespace),
						label("result", "Requeued"),
						label("tag", ""),
					),
				}))
			buildRequeuedCount := uint64(0)
			if buildRequeued != nil {
				buildRequeuedCount = *buildRequeued.Histogram.SampleCount
			}

			Expect(buildReconciliations).To(BeNumerically("==", buildReconciledCount+buildRequeuedCount))
		})

		t.Run("Build queue duration metric", func(t *testing.T) {
			var ts1, ts2 time.Time
			// The start queuing time is taken from the creation time
			ts1 = build.CreationTimestamp.Time

			// Retrieve the end queuing time from the logs
			err = NewLogWalker(&logs).
				AddStep(MatchFields(IgnoreExtras, Fields{
					"LoggerName":  Equal("camel-k.controller.build"),
					"Message":     Equal("state transition"),
					"PhaseFrom":   Equal(string(v1.BuildPhaseScheduling)),
					"PhaseTo":     Equal(string(v1.BuildPhasePending)),
					"RequestName": Equal(build.Name),
				}), func(l *LogEntry) { ts2 = l.Timestamp.Time }).
				Walk()
			Expect(err).To(BeNil())
			Expect(ts1).NotTo(BeZero())
			Expect(ts2).NotTo(BeZero())

			durationFromLogs := ts2.Sub(ts1)

			// Retrieve the queuing duration from the metric
			Expect(metrics).To(HaveKey("camel_k_build_queue_duration_seconds"))
			metric := metrics["camel_k_build_queue_duration_seconds"].Metric
			Expect(metric).To(HaveLen(1))
			histogram := metric[0].Histogram
			Expect(histogram).NotTo(BeNil())
			Expect(histogram.SampleSum).NotTo(BeNil())

			duration := *histogram.SampleSum

			// Check both durations match
			Expect(math.Abs(durationFromLogs.Seconds() - duration)).To(BeNumerically("<", 1))

			// Check the queuing duration is correctly observed in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_build_queue_duration_seconds",
				EqualP(prometheus.MetricFamily{
					Name: stringP("camel_k_build_queue_duration_seconds"),
					Help: stringP("Camel K build queue duration"),
					Type: metricTypeP(prometheus.MetricType_HISTOGRAM),
					Metric: []*prometheus.Metric{
						{
							Label: []*prometheus.LabelPair{
								label("type", "fast-jar"),
							},
							Histogram: &prometheus.Histogram{
								SampleCount: uint64P(1),
								SampleSum:   histogram.SampleSum,
								Bucket:      buckets(0, []float64{5, 15, 30, 60, 300, math.Inf(1)}),
							},
						},
					},
				}),
			))
		})

		t.Run("Integration first readiness metric", func(t *testing.T) {
			var ts1, ts2 time.Time

			// The start time is taken from the Integration status initialization timestamp
			ts1 = it.Status.InitializationTimestamp.Time
			Expect(ts1).NotTo(BeZero())
			// The end time is reported into the ready condition first truthy time
			ts2 = it.Status.GetCondition(v1.IntegrationConditionReady).FirstTruthyTime.Time
			Expect(ts2).NotTo(BeZero())

			duration := ts2.Sub(ts1)

			// Retrieve these start and end times from the logs
			err = NewLogWalker(&logs).
				AddStep(MatchFields(IgnoreExtras, Fields{
					"LoggerName":  Equal("camel-k.controller.integration"),
					"Message":     Equal("Reconciling Integration"),
					"RequestName": Equal(it.Name),
					"PhaseFrom":   Equal(string(v1.IntegrationPhaseInitialization)),
					"PhaseTo":     Equal(string(v1.IntegrationPhaseBuildingKit)),
				}), func(l *LogEntry) { ts1 = l.Timestamp.Time }).
				AddStep(MatchFields(IgnoreExtras, Fields{
					"LoggerName":  Equal("camel-k.controller.integration"),
					"Message":     HavePrefix("First readiness"),
					"RequestName": Equal(it.Name),
				}), func(l *LogEntry) { ts2 = l.Timestamp.Time }).
				Walk()
			Expect(err).To(BeNil())
			Expect(ts1).NotTo(BeZero())
			Expect(ts2).NotTo(BeZero())
			Expect(ts2).To(BeTemporally(">", ts1))
			durationFromLogs := ts2.Sub(ts1)

			// Check both durations match
			Expect(math.Abs((durationFromLogs - duration).Seconds())).To(BeNumerically("<", 1))

			// Check the duration is correctly observed in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_integration_first_readiness_seconds",
				EqualP(prometheus.MetricFamily{
					Name: stringP("camel_k_integration_first_readiness_seconds"),
					Help: stringP("Camel K integration time to first readiness"),
					Type: metricTypeP(prometheus.MetricType_HISTOGRAM),
					Metric: []*prometheus.Metric{
						{
							Histogram: &prometheus.Histogram{
								SampleCount: uint64P(1),
								SampleSum:   float64P(duration.Seconds()),
								Bucket:      buckets(duration.Seconds(), []float64{5, 10, 30, 60, 120, math.Inf(1)}),
							},
						},
					},
				}),
			))
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func getMetric(family *prometheus.MetricFamily, matcher types.GomegaMatcher) *prometheus.Metric {
	for _, metric := range family.Metric {
		if match, err := matcher.Match(metric); err != nil {
			panic(err)
		} else if match {
			return metric
		}
	}
	return nil
}

func label(name, value string) *prometheus.LabelPair {
	return &prometheus.LabelPair{
		Name:  &name,
		Value: &value,
	}
}

func bucket(value float64, upperBound float64) *prometheus.Bucket {
	var count uint64
	if value <= upperBound {
		count++
	}
	return &prometheus.Bucket{
		UpperBound:      float64P(upperBound),
		CumulativeCount: &count,
	}
}

func buckets(value float64, upperBounds []float64) []*prometheus.Bucket {
	var buckets []*prometheus.Bucket
	for _, upperBound := range upperBounds {
		buckets = append(buckets, bucket(value, upperBound))
	}
	return buckets
}

// https://prometheus.io/docs/instrumenting/exposition_formats/
func parsePrometheusData(data []byte) (map[string]*prometheus.MetricFamily, error) {
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(bytes.NewReader(data))
}

func stringP(s string) *string {
	return &s
}

func metricTypeP(t prometheus.MetricType) *prometheus.MetricType {
	return &t
}

func uint64P(i uint64) *uint64 {
	return &i
}

func float64P(f float64) *float64 {
	return &f
}
