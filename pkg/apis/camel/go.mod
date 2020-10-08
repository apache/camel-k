module github.com/apache/camel-k/pkg/apis/camel

go 1.13

require (
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	// Required to get https://github.com/kubernetes-sigs/controller-tools/pull/428
	sigs.k8s.io/controller-tools v0.0.0-20200528125929-5c0c6ae3b64b // indirect
	sigs.k8s.io/structured-merge-diff v0.0.0-20190525122527-15d366b2352e // indirect
)
