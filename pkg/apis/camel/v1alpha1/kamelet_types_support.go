package v1alpha1

import (
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConditions --
func (in *KameletStatus) GetConditions() []v1.ResourceCondition {
	res := make([]v1.ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

// GetType --
func (c KameletCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --
func (c KameletCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --
func (c KameletCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --
func (c KameletCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --
func (c KameletCondition) GetReason() string {
	return c.Reason
}

// GetMessage --
func (c KameletCondition) GetMessage() string {
	return c.Message
}
