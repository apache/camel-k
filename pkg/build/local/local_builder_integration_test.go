// +build integration

/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package local

import (
	"context"
	"testing"

	build "github.com/apache/camel-k/pkg/build/api"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {

	ctx := context.TODO()
	builder := NewLocalBuilder(ctx, test.GetTargetNamespace())

	execution := builder.Build(build.BuildSource{
		Identifier: build.BuildIdentifier{
			Name:   "test0",
			Digest: digest.Random(),
		},
		Code: build.Code{
			Content: code(),
		},
	})

	res := <-execution

	assert.Nil(t, res.Error, "Build failed")
}

func TestDoubleBuild(t *testing.T) {

	ctx := context.TODO()
	builder := NewLocalBuilder(ctx, test.GetTargetNamespace())

	execution1 := builder.Build(build.BuildSource{
		Identifier: build.BuildIdentifier{
			Name:   "test1",
			Digest: digest.Random(),
		},
		Code: build.Code{
			Content: code(),
		},
	})

	execution2 := builder.Build(build.BuildSource{
		Identifier: build.BuildIdentifier{
			Name:   "test2",
			Digest: digest.Random(),
		},
		Code: build.Code{
			Content: code(),
		},
	})

	res1 := <-execution1
	res2 := <-execution2

	assert.Nil(t, res1.Error, "Build failed")
	assert.Nil(t, res2.Error, "Build failed")
}

func TestFailedBuild(t *testing.T) {

	ctx := context.TODO()
	builder := NewLocalBuilder(ctx, test.GetTargetNamespace())

	execution := builder.Build(build.BuildSource{
		Identifier: build.BuildIdentifier{
			Name:   "test3",
			Digest: digest.Random(),
		},
		Code: build.Code{
			Content: code() + "-",
		},
	})

	res := <-execution

	assert.NotNil(t, res.Error, "Build should fail")
}

func code() string {
	return `package kamel;

import org.apache.camel.builder.RouteBuilder;

public class Routes extends RouteBuilder {

	@Override
    public void configure() throws Exception {
        from("timer:tick")
		  .to("log:info");
    }

}
`
}
