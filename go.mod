module github.com/apache/camel-k

go 1.13

require (
	github.com/Masterminds/semver v1.5.0
	github.com/apache/camel-k/pkg/apis/camel v0.0.0
	github.com/apache/camel-k/pkg/client/camel v0.0.0
	github.com/apache/camel-k/pkg/kamelet/repository v0.0.0
	github.com/container-tools/spectrum v0.3.2
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/fatih/structs v1.1.0
	github.com/gertd/go-pluralize v0.1.1
	github.com/go-logr/logr v0.1.0
	github.com/golangplus/testing v0.0.0-20180327235837-af21d9c3145e
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.1.1
	github.com/jpillora/backoff v1.0.0
	github.com/magiconair/properties v1.8.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/gomega v1.10.1
	github.com/opencontainers/runc v1.0.0-rc2.0.20190611121236-6cc515888830 // indirect
	github.com/openshift/api v3.9.1-0.20190927182313-d4a64ec2cbd8+incompatible
	github.com/operator-framework/operator-lib v0.1.0
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.42.1
	github.com/prometheus/client_golang v1.7.1
	github.com/radovskyb/watcher v1.0.6
	github.com/rs/xid v1.2.1
	github.com/scylladb/go-set v1.0.2
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stoewer/go-strcase v1.0.2
	github.com/stretchr/testify v1.5.1
	go.uber.org/multierr v1.5.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/gengo v0.0.0-20200205140755-e0e292d8aa12
	knative.dev/eventing v0.18.0
	knative.dev/pkg v0.0.0-20200922164940-4bf40ad82aab
	knative.dev/serving v0.18.0
	sigs.k8s.io/controller-runtime v0.6.3
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.9
	k8s.io/code-generator => k8s.io/code-generator v0.18.9
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
