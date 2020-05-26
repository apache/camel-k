module github.com/apache/camel-k/pkg/client/camel

go 1.13

require (
	github.com/apache/camel-k/pkg/apis/camel v0.0.0
	k8s.io/apimachinery v0.16.4
	k8s.io/client-go v0.16.4
)

// Local modules
replace github.com/apache/camel-k/pkg/apis/camel => ../../apis/camel
