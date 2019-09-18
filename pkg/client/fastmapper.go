package client

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// allowedAPIGroups contains a set of API groups that are allowed when using the fastmapper.
// Those must correspond to all groups used by the "kamel" binary tool when running out-of-cluster.
var allowedAPIGroups = map[string]bool{
	"":                          true, // core APIs
	"apiextensions.k8s.io":      true,
	"apps":                      true,
	"camel.apache.org":          true,
	"rbac.authorization.k8s.io": true,
}

// newFastDiscoveryRESTMapper comes from https://github.com/kubernetes-sigs/controller-runtime/pull/592.
// We may leverage the controller-runtime bits in the future, if that gets merged upstream.
func newFastDiscoveryRESTMapper(config *rest.Config) meta.RESTMapper {
	return meta.NewLazyRESTMapperLoader(func() (meta.RESTMapper, error) {
		return newFastDiscoveryRESTMapperWithFilter(config, func(g *metav1.APIGroup) bool {
			return allowedAPIGroups[g.Name]
		})
	})
}

func newFastDiscoveryRESTMapperWithFilter(config *rest.Config, filter func(*metav1.APIGroup) bool) (meta.RESTMapper, error) {
	dc := discovery.NewDiscoveryClientForConfigOrDie(config)
	groups, err := dc.ServerGroups()
	if err != nil {
		return nil, err
	}
	wg := wait.Group{}
	totalCount := 0
	pickedCount := 0
	var grs []*restmapper.APIGroupResources
	for _, group := range groups.Groups {
		pick := filter(&group)
		logrus.Debugf("Group: %s %v", group.Name, pick)
		totalCount++
		if !pick {
			continue
		}
		pickedCount++
		gr := &restmapper.APIGroupResources{
			Group:              group,
			VersionedResources: make(map[string][]metav1.APIResource),
		}
		grs = append(grs, gr)
		wg.Start(func() { discoverGroupResources(dc, gr) })
	}
	wg.Wait()
	logrus.Debugf("Picked %d/%d", pickedCount, totalCount)
	return restmapper.NewDiscoveryRESTMapper(grs), nil
}

func discoverGroupResources(dc discovery.DiscoveryInterface, gr *restmapper.APIGroupResources) {
	for _, version := range gr.Group.Versions {
		resources, err := dc.ServerResourcesForGroupVersion(version.GroupVersion)
		if err != nil {
			logrus.Fatal(err, version.GroupVersion)
		}
		gr.VersionedResources[version.Version] = resources.APIResources
	}
}
