// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package test

import (
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func init() {
	// Initializes the kubernetes client to auto-detect the context
	kubernetes.InitKubeClient("")

	err := install.SetupClusterwideResources()
	if err != nil {
		panic(err)
	}

	err = install.Operator(GetTargetNamespace())
	if err != nil {
		panic(err)
	}
}

func GetTargetNamespace() string {
	ns, err := kubernetes.GetClientCurrentNamespace("")
	if err != nil {
		panic(err)
	}
	return ns
}

func TimerToLogIntegrationCode() string {
	return `
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
