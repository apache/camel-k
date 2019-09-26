package kubernetes

import "strings"

// IsUnknownAPIError checks if the given error is due to some missing APIs in the cluster.
// Apparently there's no such method in Kubernetes Go API.
func IsUnknownAPIError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "no matches for kind")
}
