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
	"github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"

	prometheus "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/apache/camel-k/e2e/support/util"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestMetrics(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "java"
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"-t", "prometheus.enabled=true",
			"-t", "prometheus.service-monitor=false").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		pod := OperatorPod(ns)()
		Expect(pod).NotTo(BeNil())

		logs := StructuredLogs(ns, pod.Name, v1.PodLogOptions{})
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
				AddStep(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"LoggerName":  Equal("camel-k.controller.build"),
					"Message":     Equal("Build state transition"),
					"Phase":       Equal("Pending"),
					"RequestName": Equal(build.Name),
				}), LogEntryNoop).
				AddStep(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"LoggerName":  Equal("camel-k.controller.build"),
					"Message":     Equal("Reconciling Build"),
					"RequestName": Equal(build.Name),
				}), func(l *LogEntry) { ts1 = l.Timestamp.Time }).
				AddStep(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"LoggerName": Equal("camel-k.builder"),
					"Message":    HavePrefix("resolved image"),
				}), LogEntryNoop).
				AddStep(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"LoggerName":  Equal("camel-k.controller.build"),
					"Message":     Equal("Reconciling Build"),
					"RequestName": Equal(build.Name),
				}), func(l *LogEntry) { ts2 = l.Timestamp.Time }).
				Walk()
			Expect(err).To(BeNil())
			Expect(ts1).NotTo(BeNil())
			Expect(ts2).NotTo(BeNil())
			Expect(ts2).To(BeTemporally(">", ts1))

			durationFromLogs := ts2.Sub(ts1)
			Expect(math.Abs((durationFromLogs - duration).Seconds())).To(BeNumerically("<", 1))

			// Check the duration is observed in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_build_duration_seconds", gstruct.PointTo(Equal(prometheus.MetricFamily{
				Name: stringP("camel_k_build_duration_seconds"),
				Help: stringP("Camel K build duration"),
				Type: metricTypeP(prometheus.MetricType_HISTOGRAM),
				Metric: []*prometheus.Metric{
					{
						Label: []*prometheus.LabelPair{
							label("result", "Succeeded"),
						},
						Histogram: &prometheus.Histogram{
							SampleCount: uint64P(1),
							SampleSum:   float64P(duration.Seconds()),
							Bucket:      buckets(duration.Seconds(), []float64{30, 60, 90, 120, 300, 600, math.Inf(1)}),
						},
					},
				},
			}))))
		})

		t.Run("Build recovery attempts", func(t *testing.T) {
			// Check there are no failures reported in the Build status
			Expect(build.Status.Failure).To(BeNil())

			// Check no recovery attempts are reported in the logs
			recoveryAttemptLogged := false
			err = NewLogWalker(&logs).
				AddStep(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"LoggerName":  Equal("camel-k.controller.build"),
					"Message":     HavePrefix("Recovery attempt"),
					"Kind":        Equal("Build"),
					"RequestName": Equal(build.Name),
				}), func(l *LogEntry) { recoveryAttemptLogged = true }).
				Walk()
			Expect(err).To(BeNil())
			Expect(recoveryAttemptLogged).To(BeFalse())

			// Check no recovery attempts are observed in the corresponding metric
			Expect(metrics).To(HaveKeyWithValue("camel_k_build_recovery_attempts", gstruct.PointTo(Equal(prometheus.MetricFamily{
				Name: stringP("camel_k_build_recovery_attempts"),
				Help: stringP("Camel K build recovery attempts"),
				Type: metricTypeP(prometheus.MetricType_HISTOGRAM),
				Metric: []*prometheus.Metric{
					{
						Label: []*prometheus.LabelPair{
							label("result", "Succeeded"),
						},
						Histogram: &prometheus.Histogram{
							SampleCount: uint64P(1),
							SampleSum:   float64P(0),
							Bucket:      buckets(0, []float64{0, 1, 2, 3, 4, 5, math.Inf(1)}),
						},
					},
				},
			}))))
		})

		t.Run("Integration metrics", func(t *testing.T) {
			pod := IntegrationPod(ns, name)()
			response, err := TestClient().CoreV1().RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/q/metrics", ns, pod.Name)).DoRaw(TestContext)
			Expect(err).To(BeNil())
			assert.Contains(t, string(response), "camel.route.exchanges.total")
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
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
