package knative08compat

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1alpha1"
)

// CompatSchemeGroupVersion is the old group version used in Knative 0.8
var CompatSchemeGroupVersion = schema.GroupVersion{
	Group:   "eventing.knative.dev",
	Version: "v1alpha1",
}

// Subscription is a Knative 0.8 compatibility version for messaging.Subscription
type Subscription struct {
	messaging.Subscription
}

// SubscriptionList is a Knative 0.8 compatibility version for messaging.SubscriptionList
type SubscriptionList struct {
	messaging.SubscriptionList
}
