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

// kamel run PostgresDBAutoDatasource.java --dev 
//                                          --build-property quarkus.datasource.camel.db-kind=postgresql 
//                                          -p quarkus.datasource.camel.jdbc.url=jdbc:postgresql://postgres:5432/test 
//                                          -p quarkus.datasource.camel.username=postgresadmin 
//                                          -p quarkus.datasource.camel.password=admin123 
//                                          -d mvn:io.quarkus:quarkus-jdbc-postgresql:1.13.7.Final
// 
// Alternatively, you can bundle your credentials as a secret properties file:
//
// kubectl create secret generic my-datasource --from-file=datasource.properties
// 
// kamel run PostgresDBAutoDatasource.java --dev 
//                                          --build-property quarkus.datasource.camel.db-kind=postgresql 
//                                          --config secret:my-datasource
//                                          -d mvn:io.quarkus:quarkus-jdbc-postgresql:1.13.7.Final

import org.apache.camel.builder.RouteBuilder;

public class PostgresDBAutoDatasource extends RouteBuilder {
  @Override
  public void configure() throws Exception {
   from("timer://foo?period=10000")
   .setBody(constant("select * from test"))
   .to("jdbc:camel")
   .to("log:info");
  }

}