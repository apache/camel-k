package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// nolint: gocritic
func MirrorReadyCondition(ctx context.Context, c client.Client, it *v1.Integration) {
	if isConditionTrue(it, v1.IntegrationConditionDeploymentAvailable) || isConditionTrue(it, v1.IntegrationConditionKnativeServiceAvailable) {
		mirrorReadyConditionFromReplicaSet(ctx, c, it)
	} else if isConditionTrue(it, v1.IntegrationConditionCronJobAvailable) {
		mirrorReadyConditionFromCronJob(ctx, c, it)
	} else {
		it.Status.SetCondition(
			v1.IntegrationConditionReady,
			corev1.ConditionUnknown,
			"",
			"",
		)
	}
}

func mirrorReadyConditionFromReplicaSet(ctx context.Context, c client.Client, it *v1.Integration) {
	list := appsv1.ReplicaSetList{}
	opts := runtimeclient.MatchingLabels{
		"camel.apache.org/integration": it.Name,
	}
	if err := c.List(ctx, &list, opts, runtimeclient.InNamespace(it.Namespace)); err != nil {
		setReadyConditionError(it, err)
		return
	}

	if len(list.Items) == 0 {
		setReadyConditionError(it, errors.New("replicaset not found"))
		return
	}

	var rs *appsv1.ReplicaSet
	for _, r := range list.Items {
		r := r
		if r.Labels["camel.apache.org/generation"] == strconv.FormatInt(it.Generation, 10) {
			rs = &r
		}
	}
	if rs == nil {
		rs = &list.Items[0]
	}
	var replicas int32 = 1
	if rs.Spec.Replicas != nil {
		replicas = *rs.Spec.Replicas
	}
	if replicas == rs.Status.ReadyReplicas {
		it.Status.SetCondition(
			v1.IntegrationConditionReady,
			corev1.ConditionTrue,
			v1.IntegrationConditionReplicaSetReadyReason,
			"",
		)
	} else {
		it.Status.SetCondition(
			v1.IntegrationConditionReady,
			corev1.ConditionTrue,
			v1.IntegrationConditionReplicaSetReadyReason,
			"",
		)
	}
}

func mirrorReadyConditionFromCronJob(ctx context.Context, c client.Client, it *v1.Integration) {
	cronJob := v1beta1.CronJob{}
	if err := c.Get(ctx, runtimeclient.ObjectKey{Namespace: it.Namespace, Name: it.Name}, &cronJob); err != nil {
		setReadyConditionError(it, err)
	} else {
		// CronJob status is not tracked by Kubernetes
		it.Status.SetCondition(
			v1.IntegrationConditionReady,
			corev1.ConditionTrue,
			v1.IntegrationConditionCronJobCreatedReason,
			"",
		)
	}
}

func isConditionTrue(it *v1.Integration, conditionType v1.IntegrationConditionType) bool {
	cond := it.Status.GetCondition(conditionType)
	if cond == nil {
		return false
	}
	return cond.Status == corev1.ConditionTrue
}

func setReadyConditionError(it *v1.Integration, err error) {
	it.Status.SetCondition(
		v1.IntegrationConditionReady,
		corev1.ConditionUnknown,
		v1.IntegrationConditionErrorReason,
		fmt.Sprintf("%v", err),
	)
}
