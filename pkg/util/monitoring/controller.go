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

package monitoring

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/prometheus/client_golang/prometheus"
)

type resultLabelValue string

const (
	reconciled resultLabelValue = "Reconciled"
	errored    resultLabelValue = "Errored"
	requeued   resultLabelValue = "Requeued"

	namespaceLabel = "namespace"
	groupLabel     = "group"
	versionLabel   = "version"
	kindLabel      = "kind"
	resultLabel    = "result"
	tagLabel       = "tag"
)

type tagLabelValue string

const (
	platformError tagLabelValue = "PlatformError"
	userError     tagLabelValue = "UserError"
)

type instrumentedReconciler struct {
	reconciler reconcile.Reconciler
	gvk        schema.GroupVersionKind
}

var _ reconcile.Reconciler = &instrumentedReconciler{}

func NewInstrumentedReconciler(rec reconcile.Reconciler, gvk schema.GroupVersionKind) reconcile.Reconciler {
	return &instrumentedReconciler{
		reconciler: rec,
		gvk:        gvk,
	}
}

func (r *instrumentedReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	timer := NewTimer()

	res, err := r.reconciler.Reconcile(ctx, request)

	labels := prometheus.Labels{
		namespaceLabel: request.Namespace,
		groupLabel:     r.gvk.Group,
		versionLabel:   r.gvk.Version,
		kindLabel:      r.gvk.Kind,
		resultLabel:    resultLabelFor(res, err),
		tagLabel:       "",
	}
	if err != nil {
		// Controller errors are tagged as platform errors
		labels[tagLabel] = string(platformError)
	}

	timer.ObserveDurationInSeconds(loopDuration.With(labels))

	return res, err
}

func resultLabelFor(res reconcile.Result, err error) string {
	var label resultLabelValue
	if err != nil {
		label = errored
	} else if res.Requeue || res.RequeueAfter > 0 {
		label = requeued
	} else {
		label = reconciled
	}
	return string(label)
}

var (
	loopDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "camel_k_reconciliation_duration_seconds",
			Help: "Camel K reconciliation loop duration",
			Buckets: []float64{
				0.25 * time.Second.Seconds(),
				0.5 * time.Second.Seconds(),
				1 * time.Second.Seconds(),
				5 * time.Second.Seconds(),
			},
		},
		[]string{
			namespaceLabel,
			groupLabel,
			versionLabel,
			kindLabel,
			resultLabel,
			tagLabel,
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(loopDuration)
}
