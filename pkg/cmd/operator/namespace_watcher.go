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

package operator

import (
	"context"
	"sort"
	"time"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

const (
	// namespaceWatcherDebounce is how long the watcher waits after a namespace event before
	// re-evaluating, so that a burst of events (e.g. labelling many namespaces at once) results
	// in a single operator restart instead of one per event.
	namespaceWatcherDebounce = 5 * time.Second
	// namespaceWatcherResyncInterval is the safety-net period at which the watcher recomputes the
	// desired namespace set even without an event. It lets an out-of-order setup (namespace
	// labelled before its RBAC was installed) self-heal without manual intervention.
	namespaceWatcherResyncInterval = 2 * time.Minute
)

// namespaceWatcher watches Namespace objects matching a label selector cluster-wide and requests
// a graceful operator restart whenever the set of namespaces the operator should (and can) watch
// changes. controller-runtime does not allow adding or removing namespaces from a running cache,
// so reconfiguration is achieved by restarting: the operator recomputes its watched set at
// startup. The watcher only ever reads Namespace objects (metadata), never workload resources.
type namespaceWatcher struct {
	// config is the in-cluster REST config used to build the cluster-scoped Namespace cache.
	config *rest.Config
	// scheme must know corev1.Namespace.
	scheme *runtime.Scheme
	// reviewer issues SelfSubjectAccessReviews to pre-flight per-namespace access.
	reviewer ctrl.Client
	// selector selects which namespaces are candidates for watching.
	selector labels.Selector
	// operatorNamespace is always watched and never subject to dynamic removal.
	operatorNamespace string
	// staticNamespaces are namespaces explicitly configured via WATCH_NAMESPACE.
	staticNamespaces []string
	// current is the immutable set of namespaces the running manager was started with. When the
	// freshly computed desired set differs from this, a restart is requested.
	current map[string]bool
	// requestRestart cancels the manager context to trigger a graceful restart.
	requestRestart context.CancelFunc

	debounce time.Duration
	resync   time.Duration
}

// Start implements manager.Runnable. It blocks until the context is cancelled or a restart is
// requested. It is leader-election gated by default (mgr.Add treats non-cache runnables as
// requiring leadership), so only the active operator instance watches namespaces and triggers
// restarts.
func (w *namespaceWatcher) Start(ctx context.Context) error {
	debounce := w.debounce
	if debounce <= 0 {
		debounce = namespaceWatcherDebounce
	}
	resync := w.resync
	if resync <= 0 {
		resync = namespaceWatcherResyncInterval
	}

	nsCache, err := cache.New(w.config, cache.Options{
		Scheme: w.scheme,
		ByObject: map[ctrl.Object]cache.ByObject{
			// Namespace is cluster-scoped: leave Namespaces nil, filter by label only.
			&corev1.Namespace{}: {Label: w.selector},
		},
	})
	if err != nil {
		return err
	}

	informer, err := nsCache.GetInformer(ctx, &corev1.Namespace{})
	if err != nil {
		return err
	}

	trigger := make(chan struct{}, 1)
	ping := func() {
		select {
		case trigger <- struct{}{}:
		default:
		}
	}
	// In a label-filtered informer, gaining the label surfaces as an Add and losing it (or the
	// namespace being deleted) surfaces as a Delete. Both may change the watched set.
	if _, err := informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { ping() },
		DeleteFunc: func(_ interface{}) { ping() },
	}); err != nil {
		return err
	}

	cacheCtx, cacheCancel := context.WithCancel(ctx)
	defer cacheCancel()
	go func() {
		if startErr := nsCache.Start(cacheCtx); startErr != nil {
			log.Error(startErr, "namespace watcher cache stopped with an error")
		}
	}()
	if !nsCache.WaitForCacheSync(cacheCtx) {
		// Context cancelled before the cache synced; nothing more to do.
		return nil
	}

	log.Info("Namespace watcher started", "selector", w.selector.String(),
		"watching", sortedKeys(w.current))

	ticker := time.NewTicker(resync)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			ping()
		case <-trigger:
			// Debounce to coalesce bursts of namespace events into a single evaluation.
			if !sleep(ctx, debounce) {
				return nil
			}
			drain(trigger)
			if w.changed(ctx) {
				log.Info("Watched namespace set changed, restarting the operator to apply it")
				w.requestRestart()

				return nil
			}
		}
	}
}

// changed recomputes the desired (accessible) namespace set and reports whether it differs from
// the set the operator is currently running with. Transient API errors are treated as "no change"
// to avoid restart loops on momentary connectivity issues.
func (w *namespaceWatcher) changed(ctx context.Context) bool {
	desired, err := computeWatchedNamespaces(ctx, w.reviewer, w.config, w.scheme,
		w.operatorNamespace, w.staticNamespaces, w.selector)
	if err != nil {
		log.Error(err, "could not recompute the watched namespace set; keeping the current one")

		return false
	}

	if len(desired) != len(w.current) {
		return true
	}
	for ns := range desired {
		if !w.current[ns] {
			return true
		}
	}

	return false
}

// computeWatchedNamespaces returns the set of namespaces the operator should watch: the operator
// namespace, the statically configured namespaces, and (when a selector is provided) every
// namespace matching the selector. Namespaces other than the operator namespace are included only
// if the operator actually has permission to watch Integrations there, so that a namespace lacking
// the required RBAC is skipped (with a warning) rather than blocking the manager cache from syncing.
func computeWatchedNamespaces(
	ctx context.Context,
	reviewer ctrl.Client,
	config *rest.Config,
	scheme *runtime.Scheme,
	operatorNamespace string,
	staticNamespaces []string,
	selector labels.Selector,
) (map[string]bool, error) {
	candidates := make(map[string]bool)
	if operatorNamespace != "" {
		candidates[operatorNamespace] = true
	}
	for _, ns := range staticNamespaces {
		candidates[ns] = true
	}

	if selector != nil {
		discoverer := reviewer
		if discoverer == nil {
			c, err := ctrl.New(config, ctrl.Options{Scheme: scheme})
			if err != nil {
				return nil, err
			}
			discoverer = c
		}
		list := &corev1.NamespaceList{}
		if err := discoverer.List(ctx, list, ctrl.MatchingLabelsSelector{Selector: selector}); err != nil {
			return nil, err
		}
		for i := range list.Items {
			candidates[list.Items[i].Name] = true
		}
	}

	watched := make(map[string]bool, len(candidates))
	for ns := range candidates {
		// The operator namespace is always watched; the operator must be able to operate there.
		if ns == operatorNamespace {
			watched[ns] = true

			continue
		}
		allowed, err := canWatchIntegrations(ctx, reviewer, ns)
		if err != nil {
			// If we cannot determine access, be conservative and skip the namespace so a
			// permission problem cannot block the whole cache from syncing.
			log.Error(err, "could not verify operator access to namespace; skipping it", "namespace", ns)

			continue
		}
		if !allowed {
			log.Info("Skipping namespace: operator lacks RBAC to watch Integrations there "+
				"(install the namespaced Role/RoleBinding to enable it)", "namespace", ns)

			continue
		}
		watched[ns] = true
	}

	return watched, nil
}

// canWatchIntegrations reports whether the operator's ServiceAccount may watch Integrations in the
// given namespace, using a SelfSubjectAccessReview (which any authenticated principal may create).
func canWatchIntegrations(ctx context.Context, reviewer ctrl.Client, namespace string) (bool, error) {
	review := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Group:     v1.SchemeGroupVersion.Group,
				Resource:  "integrations",
				Verb:      "watch",
			},
		},
	}
	if err := reviewer.Create(ctx, review); err != nil {
		return false, err
	}

	return review.Status.Allowed, nil
}

// sortedKeys returns the keys of the set sorted, for stable logging.
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

// sleep waits for d or until ctx is done. It returns false if the context was cancelled.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// drain empties any pending value from the trigger channel.
func drain(ch <-chan struct{}) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
