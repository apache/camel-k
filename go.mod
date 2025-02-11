module github.com/apache/camel-k/v2

go 1.23

require (
	github.com/Masterminds/semver v1.5.0
	github.com/container-tools/spectrum v0.6.68
	github.com/evanphx/json-patch v5.9.11+incompatible
	github.com/fsnotify/fsnotify v1.8.0
	github.com/gertd/go-pluralize v0.2.1
	github.com/go-logr/logr v1.4.2
	github.com/go-viper/mapstructure/v2 v2.2.1
	github.com/google/go-github/v52 v52.0.0
	github.com/google/uuid v1.6.0
	github.com/imdario/mergo v0.3.16
	github.com/jpillora/backoff v1.0.0
	github.com/magiconair/properties v1.8.9
	github.com/mattn/go-shellwords v1.0.12
	github.com/onsi/gomega v1.36.2
	// go get github.com/openshift/api@release-4.15
	github.com/openshift/api v0.0.0-20240228005710-4511c790cc60
	github.com/operator-framework/api v0.26.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.76.2
	github.com/prometheus/client_golang v1.20.5
	github.com/prometheus/client_model v0.6.1
	github.com/prometheus/common v0.62.0
	github.com/redhat-developer/service-binding-operator v1.4.0
	github.com/rs/xid v1.6.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.19.0
	github.com/stoewer/go-strcase v1.3.0
	github.com/stretchr/testify v1.10.0
	go.uber.org/automaxprocs v1.6.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/oauth2 v0.26.0
	golang.org/x/sync v0.11.0
	golang.org/x/term v0.29.0
	golang.org/x/text v0.22.0
	golang.org/x/time v0.10.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.30.8
	k8s.io/apiextensions-apiserver v0.30.8
	k8s.io/apimachinery v0.30.8
	k8s.io/cli-runtime v0.30.8
	k8s.io/client-go v0.30.8
	k8s.io/gengo v0.0.0-20240404160639-a0386bf69313
	k8s.io/klog/v2 v2.130.1
	k8s.io/kubectl v0.30.8
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8
	knative.dev/eventing v0.43.3
	knative.dev/pkg v0.0.0-20241021183759-9b9d535af5ad
	knative.dev/serving v0.43.1
	sigs.k8s.io/controller-runtime v0.18.5
	sigs.k8s.io/structured-merge-diff/v4 v4.5.0
)

require (
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.4.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230217124315-7d5c6f04bbb8 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/blendle/zapdriver v1.3.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudevents/sdk-go/sql/v2 v2.15.2 // indirect
	github.com/cloudevents/sdk-go/v2 v2.15.2 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/cli v27.4.1+incompatible // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-containerregistry v0.20.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.21.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/rickb777/date v1.13.0 // indirect
	github.com/rickb777/plural v1.2.1 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/vbatts/tar-split v0.11.3 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/exp v0.0.0-20240808152545-0cdaa3abc0fa // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/api v0.183.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240808142205-8e686545bdb8 // indirect
	knative.dev/networking v0.0.0-20241022012959-60e29ff520dc // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
