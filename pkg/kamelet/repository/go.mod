module github.com/apache/camel-k/pkg/kamelet/repository

go 1.13

require (
	github.com/apache/camel-k/pkg/apis/camel v0.0.0
	github.com/apache/camel-k/pkg/client/camel v0.0.0
	github.com/google/go-github/v32 v32.1.0
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7
	github.com/stretchr/testify v1.5.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	k8s.io/apimachinery v0.19.8
)

// Local modules
replace github.com/apache/camel-k/pkg/apis/camel => ../../apis/camel

replace github.com/apache/camel-k/pkg/client/camel => ../../client/camel
