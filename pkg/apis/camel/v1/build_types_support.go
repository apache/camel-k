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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewBuild(namespace string, name string) *Build {
	return &Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func NewBuildList() BuildList {
	return BuildList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       BuildKind,
		},
	}
}

// BuilderPodNamespace returns the namespace of the operator in charge to reconcile this Build.
func (build *Build) BuilderPodNamespace() string {
	for _, t := range build.Spec.Tasks {
		if t.Builder != nil {
			return t.Builder.Configuration.BuilderPodNamespace
		}
	}
	return ""
}

// BuilderConfiguration returns the builder configuration for this Build.
func (build *Build) BuilderConfiguration() *BuildConfiguration {
	return BuilderConfigurationTasks(build.Spec.Tasks)

}

// BuilderConfigurationTasks returns the builder configuration from the task list.
func BuilderConfigurationTasks(tasks []Task) *BuildConfiguration {
	for _, t := range tasks {
		if t.Builder != nil {
			return &t.Builder.Configuration
		}
	}
	return &BuildConfiguration{}
}

// SetBuilderConfiguration set the configuration required for this Build.
func (build *Build) SetBuilderConfiguration(conf *BuildConfiguration) {
	SetBuilderConfigurationTasks(build.Spec.Tasks, conf)
}

// SetBuilderConfigurationTasks set the configuration required for the builder in the list of tasks.
func SetBuilderConfigurationTasks(tasks []Task, conf *BuildConfiguration) {
	for _, t := range tasks {
		if t.Builder != nil {
			t.Builder.Configuration = *conf
			return
		}
	}
}

func (buildPhase *BuildPhase) String() string {
	return string(*buildPhase)
}

// GetCondition returns the condition with the provided type.
func (in *BuildStatus) GetCondition(condType BuildConditionType) *BuildCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func (in *BuildStatus) Failed(err error) BuildStatus {
	in.Error = err.Error()
	in.Phase = BuildPhaseFailed
	return *in
}

func (in *BuildStatus) SetCondition(condType BuildConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(BuildCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (in *BuildStatus) SetErrorCondition(condType BuildConditionType, reason string, err error) {
	in.SetConditions(BuildCondition{
		Type:               condType,
		Status:             corev1.ConditionFalse,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	})
}

// SetConditions updates the resource to include the provided conditions.
//
// If a condition that we are about to add already exists and has the same status and
// reason then we are not going to update.
func (in *BuildStatus) SetConditions(conditions ...BuildCondition) {
	for _, condition := range conditions {
		if condition.LastUpdateTime.IsZero() {
			condition.LastUpdateTime = metav1.Now()
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.Now()
		}

		currentCond := in.GetCondition(condition.Type)

		if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
			return
		}
		// Do not update lastTransitionTime if the status of the condition doesn't change.
		if currentCond != nil && currentCond.Status == condition.Status {
			condition.LastTransitionTime = currentCond.LastTransitionTime
		}

		in.RemoveCondition(condition.Type)
		in.Conditions = append(in.Conditions, condition)
	}
}

// RemoveCondition removes the resource condition with the provided type.
func (in *BuildStatus) RemoveCondition(condType BuildConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

var _ ResourceCondition = BuildCondition{}

func (in *BuildStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

func (c BuildCondition) GetType() string {
	return string(c.Type)
}

func (c BuildCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

func (c BuildCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

func (c BuildCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

func (c BuildCondition) GetReason() string {
	return c.Reason
}

func (c BuildCondition) GetMessage() string {
	return c.Message
}

func (bl BuildList) HasRunningBuilds() bool {
	for _, b := range bl.Items {
		if b.Status.Phase == BuildPhasePending || b.Status.Phase == BuildPhaseRunning {
			return true
		}
	}

	return false
}

func (bl BuildList) HasScheduledBuildsBefore(build *Build) bool {
	for _, b := range bl.Items {
		if b.Name == build.Name {
			continue
		}

		if (b.Status.Phase == BuildPhaseInitialization || b.Status.Phase == BuildPhaseScheduling) &&
			b.CreationTimestamp.Before(&build.CreationTimestamp) {
			return true
		}
	}

	return false
}
