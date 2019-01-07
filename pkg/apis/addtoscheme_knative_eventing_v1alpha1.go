package apis

import (
	eventing "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, eventing.AddToScheme)
}
