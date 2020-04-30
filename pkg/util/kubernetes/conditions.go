package kubernetes

import (
	"context"
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// nolint: gocritic
func MirrorReadyCondition(ctx context.Context, c client.Client, it *v1.Integration) {
	if isConditionTrue(it, v1.IntegrationConditionDeploymentAvailable) {
		mirrorReadyConditionFromDeployment(ctx, c, it)
	} else if isConditionTrue(it, v1.IntegrationConditionKnativeServiceAvailable) {
		mirrorReadyConditionFromKnativeService(ctx, c, it)
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

func mirrorReadyConditionFromDeployment(ctx context.Context, c client.Client, it *v1.Integration) {
	deployment := appsv1.Deployment{}
	if err := c.Get(ctx, runtimeclient.ObjectKey{Namespace: it.Namespace, Name: it.Name}, &deployment); err != nil {
		setReadyConditionError(it, err)
	} else {
		for _, c := range deployment.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable {
				it.Status.SetCondition(
					v1.IntegrationConditionReady,
					c.Status,
					c.Reason,
					c.Message,
				)
			}
		}
	}
}

func mirrorReadyConditionFromKnativeService(ctx context.Context, c client.Client, it *v1.Integration) {
	service := servingv1.Service{}
	if err := c.Get(ctx, runtimeclient.ObjectKey{Namespace: it.Namespace, Name: it.Name}, &service); err != nil {
		setReadyConditionError(it, err)
	} else {
		for _, c := range service.Status.Conditions {
			if c.Type == apis.ConditionReady {
				it.Status.SetCondition(
					v1.IntegrationConditionReady,
					c.Status,
					c.Reason,
					c.Message,
				)
			}
		}
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
