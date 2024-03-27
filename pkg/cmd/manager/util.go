package manager

import (
	"context"
	"fmt"
	"os"

	"github.com/apache/camel-k/v2/pkg/platform"
	logutil "github.com/apache/camel-k/v2/pkg/util/log"
	"go.uber.org/automaxprocs/maxprocs"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// setMaxprocs set go maxprocs according to the container environment.
func setMaxprocs(log logutil.Logger) (string, error) {
	_, err := maxprocs.Set(maxprocs.Logger(func(f string, a ...interface{}) { log.Info(fmt.Sprintf(f, a)) }))

	return "failed to set GOMAXPROCS from cgroups", err
}

// setOperatorImage set the operator container image if it runs in-container.
func setOperatorImage(ctx context.Context, bootstrapClient ctrl.Client, controllerNamespace string) (string, error) {
	var err error
	platform.OperatorImage, err = getOperatorImage(controllerNamespace, platform.GetOperatorPodName(), ctx, bootstrapClient)
	return "cannot get operator container image", err
}

// getOperatorImage returns the image currently used by the running operator if present (when running out of cluster, it may be absent).
func getOperatorImage(namespace string, podName string, ctx context.Context, c ctrl.Reader) (string, error) {
	if namespace == "" || podName == "" {
		return "", nil
	}

	pod := corev1.Pod{}
	if err := c.Get(ctx, ctrl.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil && k8serrors.IsNotFound(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	if len(pod.Spec.Containers) == 0 {
		return "", fmt.Errorf("no containers found in operator pod")
	}
	return pod.Spec.Containers[0].Image, nil
}

// GetWatchNamespace returns the Namespace the operator should be watching for changes.
func GetWatchNamespace(watchNamespaceEnvVar string) (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}
