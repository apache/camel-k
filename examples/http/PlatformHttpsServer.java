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

// Self signed certificate generation:
//
// openssl genrsa -out server.key 2048
// openssl req -new -key server.key -out server.csr
// openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt

// Storing certificate and keys in a secret
// kubectl create secret generic my-self-signed-ssl --from-file=server.key --from-file=server.crt

// Integration execution
//
// kamel run PlatformHttpsServer.java -p quarkus.http.ssl.certificate.file=/etc/ssl/my-self-signed-ssl/server.crt \
//                                    -p quarkus.http.ssl.certificate.key-file=/etc/ssl/my-self-signed-ssl/server.key \ 
//                                    --resource secret:my-self-signed-ssl@/etc/ssl/my-self-signed-ssl \
//                                    -t container.port=8443 --dev

// kamel run PlatformHttpsServer.java -p quarkus.http.ssl.certificate.file=/etc/ssl/my-self-signed-ssl/server.crt -p quarkus.http.ssl.certificate.key-file=/etc/ssl/my-self-signed-ssl/server.key --resource secret:my-self-signed-ssl@/etc/ssl/my-self-signed-ssl -t container.port=8443 --dev

// Test
//
// recover the service location. If you're running on minikube, minikube service platform-https-server --url=true
// curl -H "name:World" -k http://<service-location>/hello
//

import org.apache.camel.builder.RouteBuilder;

public class PlatformHttpsServer extends RouteBuilder {
  @Override
  public void configure() throws Exception {
    from("platform-http:/hello?httpMethodRestrict=GET").setBody(simple("Hello ${header.name}"));
  }
}