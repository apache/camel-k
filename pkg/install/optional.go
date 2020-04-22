package install

import (
	"context"

	"github.com/apache/camel-k/pkg/client"
	"github.com/go-logr/logr"
)

// OperatorStartupOptionalTools tries to install optional tools at operator startup and warns if something goes wrong
func OperatorStartupOptionalTools(ctx context.Context, c client.Client, log logr.Logger) {

	// Try to register the OpenShift CLI Download link if possible
	if err := OpenShiftConsoleDownloadLink(ctx, c); err != nil {
		log.Info("Cannot install OpenShift CLI download link: skipping.")
		log.V(8).Info("Error while installing OpenShift CLI download link", "error", err)
	}

	// Try to register the cluster role for standard admin and edit users
	if clusterRoleInstalled, err := IsClusterRoleInstalled(ctx, c); err != nil {
		log.Info("Cannot detect user cluster role: skipping.")
		log.V(8).Info("Error while getting user cluster role", "error", err)
	} else if !clusterRoleInstalled {
		if err := installClusterRole(ctx, c, nil); err != nil {
			log.Info("Cannot install user cluster role: skipping.")
			log.V(8).Info("Error while installing user cluster role", "error", err)
		}
	}

}
