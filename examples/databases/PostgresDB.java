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

// You can use the sample postgres database available at /postgres-deploy/README.md
//
// kamel run PostgresDB.java --dev -d mvn:org.postgresql:postgresql:42.2.21 -d mvn:org.apache.commons:commons-dbcp2:2.8.0

import org.apache.camel.builder.RouteBuilder;
import org.apache.commons.dbcp2.BasicDataSource;

public class PostgresDB extends RouteBuilder {
  @Override
  public void configure() throws Exception {
   registerDatasource();

   from("timer://foo?period=10000")
   .setBody(constant("select * from test"))
   .to("jdbc:myPostgresDS")
   .to("log:info");
  }

  private void registerDatasource() throws Exception {
   BasicDataSource ds = new BasicDataSource();
   ds.setUsername("postgresadmin");
   ds.setDriverClassName("org.postgresql.Driver");
   ds.setPassword("admin123");
   ds.setUrl("jdbc:postgresql://postgres:5432/test");

   this.getContext().getRegistry().bind("myPostgresDS", ds);
 }

}