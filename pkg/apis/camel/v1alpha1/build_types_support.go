package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// NewBuild --
func NewBuild(namespace string, name string) Build {
	return Build{
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

// NewBuildList --
func NewBuildList() BuildList {
	return BuildList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       BuildKind,
		},
	}
}
