module github.com/apache/camel-k

require (
	cloud.google.com/go v0.43.0 // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/alecthomas/jsonschema v0.0.0-20190122210438-a6952de1bbe6
	github.com/coreos/prometheus-operator v0.29.0
	github.com/fatih/structs v1.1.0
	github.com/go-logr/logr v0.1.0
	github.com/google/go-containerregistry v0.0.0-20190206233756-dbc4da98389f // indirect
	github.com/google/uuid v1.1.1
	github.com/jpillora/backoff v0.0.0-20170918002102-8eab2debe79d
	github.com/knative/eventing v0.7.1
	github.com/knative/pkg v0.0.0-20190624141606-d82505e6c5b4
	github.com/knative/serving v0.7.1
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/operator-framework/operator-sdk v0.9.0
	github.com/pkg/errors v0.8.1
	github.com/radovskyb/watcher v1.0.6
	github.com/rs/xid v1.2.1
	github.com/scylladb/go-set v1.0.2
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/cobra v0.0.3
	github.com/stoewer/go-strcase v1.0.2
	github.com/stretchr/testify v1.3.0
	go.uber.org/multierr v1.1.0
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	golang.org/x/sys v0.0.0-20190712062909-fae7ac547cb7 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190612125737-db0771252981
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190510232812-a01b7d5d6c22 // indirect
	sigs.k8s.io/controller-runtime v0.1.10
)

// Pinned to operator-sdk 0.9.0 / kubernetes 1.13.4
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190228174905-79427f02047f
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190228180923-a9e421a79326
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190228175259-3e0149950b0e
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20180711000925-0cf8f7e6ed1d
	k8s.io/kubernetes => k8s.io/kubernetes v1.13.4
)
