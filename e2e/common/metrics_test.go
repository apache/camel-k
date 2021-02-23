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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"

	prometheus "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestMetrics(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "java"
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"-t", "prometheus.enabled=true",
			"-t", "prometheus.service-monitor=false").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(v1.PodRunning))
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

		t.Run("Build duration metric", func(t *testing.T) {
			it := Integration(ns, name)()
			Expect(it).NotTo(BeNil())
			build := Build(ns, it.Status.Kit)()
			Expect(build).NotTo(BeNil())

			// Get the duration from the Build status
			duration, err := time.ParseDuration(build.Status.Duration)
			Expect(err).To(BeNil())

			// Check it's consistent with the duration observed from logs
			durationFromLogs := buildDuration(&logs, ns, build.Name)
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
							Bucket: []*prometheus.Bucket{
								bucket(duration, 30),
								bucket(duration, 60),
								bucket(duration, 90),
								bucket(duration, 120),
								bucket(duration, 300),
								bucket(duration, 600),
								bucket(duration, math.Inf(1)),
							},
						},
					},
				},
			}))))
		})

		t.Run("Integration metrics", func(t *testing.T) {
			pod := IntegrationPod(ns, name)()
			response, err := TestClient().CoreV1().RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/metrics", ns, pod.Name)).DoRaw(TestContext)
			Expect(err).To(BeNil())
			assert.Contains(t, string(response), "camel.route.exchanges.total")
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func buildDuration(logs *[]LogEntry, ns, buildName string) time.Duration {
	var ts1, ts2 time.Time
	for _, log := range *logs {
		if ts1.IsZero() &&
			log.LoggerName == "camel-k.controller.build" &&
			log.Message == "Build state transition" &&
			log.RequestNamespace == ns &&
			log.RequestName == buildName &&
			log.Phase == "Pending" {
			ts1 = log.Timestamp.Time
		}
		if ts2.IsZero() &&
			log.LoggerName == "camel-k.builder" &&
			strings.HasPrefix(log.Message, "resolved image") {
			ts2 = log.Timestamp.Time
		}
	}
	return ts2.Sub(ts1)
}

func label(name, value string) *prometheus.LabelPair {
	return &prometheus.LabelPair{
		Name:  &name,
		Value: &value,
	}
}

func bucket(duration time.Duration, boundSeconds float64) *prometheus.Bucket {
	var count uint64
	if duration.Seconds() < boundSeconds {
		count++
	}
	return &prometheus.Bucket{
		UpperBound:      float64P(boundSeconds),
		CumulativeCount: &count,
	}
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
