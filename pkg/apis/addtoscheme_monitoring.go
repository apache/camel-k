package apis

import "github.com/apache/camel-k/pkg/util/monitoring"

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, monitoring.AddToScheme)
}
