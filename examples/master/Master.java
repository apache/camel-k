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

// camel-k: language=java

import org.apache.camel.builder.RouteBuilder;

/**
 * This example shows how to start a route on a single instance of the integration.
 * Increase the number of replicas to see it in action (the route will be started on a single pod only).
 */
public class Master extends RouteBuilder {
  @Override
  public void configure() throws Exception {

      // Write your routes here, for example:
      from("master:lock:timer:master?period=1000")
        .setBody()
          .simple("This message is printed by a single pod, even if you increase the number of replicas!")
        .to("log:info");

  }
}
