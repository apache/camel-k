/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Create a configmap holding a jar in order to simulate the presence of a dependency on the runtime image
// kubectl create configmap my-dep --from-file=sample-1.0.jar

//kamel run --resource configmap:my-dep -t jvm.classpath=/etc/camel/resources/my-dep/sample-1.0.jar Classpath.java --dev

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.example.MyClass;

public class Classpath extends RouteBuilder {
  @Override
  public void configure() throws Exception {
	  from("timer:tick")
        .log(MyClass.sayHello());
  }
}