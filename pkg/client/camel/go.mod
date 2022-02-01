module github.com/apache/camel-k/pkg/client/camel

go 1.15

require (
	github.com/apache/camel-k/pkg/apis/camel v0.0.0
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.io/code-generator v0.22.5 // indirect
)

// Local modules
replace github.com/apache/camel-k/pkg/apis/camel => ../../apis/camel
