module github.com/apache/camel-k/pkg/client/camel

go 1.13

require (
	github.com/apache/camel-k/pkg/apis/camel v0.0.0
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
)

replace (
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
)

// Local modules
replace github.com/apache/camel-k/pkg/apis/camel => ../../apis/camel
