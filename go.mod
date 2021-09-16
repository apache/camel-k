module github.com/apache/camel-k

go 1.15

require (
	github.com/Masterminds/semver v1.5.0
	github.com/Microsoft/hcsshim v0.8.15 // indirect
	github.com/apache/camel-k/pkg/apis/camel v0.0.0
	github.com/apache/camel-k/pkg/client/camel v0.0.0
	github.com/apache/camel-k/pkg/kamelet/repository v0.0.0
	github.com/container-tools/spectrum v0.3.4
	github.com/containerd/continuity v0.0.0-20210208174643-50096c924a4e // indirect
	github.com/evanphx/json-patch v4.11.0+incompatible
	github.com/fatih/structs v1.1.0
	github.com/gertd/go-pluralize v0.1.1
	github.com/go-logr/logr v0.4.0
	github.com/golangplus/testing v1.0.0
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.2.0
	github.com/jpillora/backoff v1.0.0
	github.com/magiconair/properties v1.8.5
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/gomega v1.10.4
	github.com/openshift/api v3.9.1-0.20190927182313-d4a64ec2cbd8+incompatible
	github.com/operator-framework/api v0.3.8
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.42.1
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.20.0
	github.com/radovskyb/watcher v1.0.6
	github.com/redhat-developer/service-binding-operator v0.8.0
	github.com/rs/xid v1.2.1
	github.com/scylladb/go-set v1.0.2
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stoewer/go-strcase v1.2.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/oauth2 v0.0.0-20210413134643-5e61552d6c78
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.22.2
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027
	k8s.io/klog/v2 v2.9.0
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	knative.dev/eventing v0.23.2
	knative.dev/pkg v0.0.0-20210510175900-4564797bf3b7
	knative.dev/serving v0.23.1
	sigs.k8s.io/controller-runtime v0.8.3
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm

// Using a fork that removes the https ping before using http in case of insecure registry (for Spectrum)
replace github.com/google/go-containerregistry => github.com/nicolaferraro/go-containerregistry v0.0.0-20200428072705-e7aced86aca8

// Local modules
replace (
	github.com/apache/camel-k/pkg/apis/camel => ./pkg/apis/camel
	github.com/apache/camel-k/pkg/client/camel => ./pkg/client/camel
	github.com/apache/camel-k/pkg/kamelet/repository => ./pkg/kamelet/repository
)
