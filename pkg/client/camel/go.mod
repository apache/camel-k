module github.com/apache/camel-k/pkg/client/camel

go 1.15

require (
	github.com/apache/camel-k/pkg/apis/camel v1.5.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/code-generator v0.21.1 // indirect
)

// Local modules
replace github.com/apache/camel-k/pkg/apis/camel => ../../apis/camel
