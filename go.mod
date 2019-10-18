module github.com/apache/camel-k

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.2 // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/alecthomas/jsonschema v0.0.0-20190122210438-a6952de1bbe6
	github.com/coreos/prometheus-operator v0.29.0
	github.com/fatih/structs v1.1.0
	github.com/go-logr/logr v0.1.0
	github.com/google/go-containerregistry v0.0.0-20190206233756-dbc4da98389f // indirect
	github.com/google/uuid v1.1.1
	github.com/jpillora/backoff v0.0.0-20170918002102-8eab2debe79d
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/gomega v1.5.0
	github.com/openshift/api v0.0.0-20190927182313-d4a64ec2cbd8+incompatible
	github.com/operator-framework/operator-sdk v0.11.0
	github.com/pkg/errors v0.8.1
	github.com/radovskyb/watcher v1.0.6
	github.com/rs/xid v1.2.1
	github.com/scylladb/go-set v1.0.2
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/stoewer/go-strcase v1.0.2
	github.com/stretchr/testify v1.3.0
	go.uber.org/multierr v1.1.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	knative.dev/eventing v0.9.0
	knative.dev/pkg v0.0.0-20191017202117-b5a8deb92e5c
	knative.dev/serving v0.9.0
	sigs.k8s.io/controller-runtime v0.2.0
)

// Pinned to kubernetes 1.15.3:
// - Knative 0.9.0 requires 1.15.3
// - Operator SDK requires 1.14.1
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190819143637-0dbe462fe92d // indirect
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20181213153952-835b10687cb6 // indirect
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab // indirect
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190819145148-d91c85d212d5
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190311093542-50b561225d70 // indirect
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a // indirect
	k8s.io/helm => k8s.io/helm v2.14.1+incompatible // indirect
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190510232812-a01b7d5d6c22 // indirect
	k8s.io/kube-state-metrics => k8s.io/kube-state-metrics v1.7.2 // indirect
)

// Indirect operator-sdk dependencies use git.apache.org, which is frequently
// down. The github mirror should be used instead.
// Locking to a specific version (from 'go mod graph'):
replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
