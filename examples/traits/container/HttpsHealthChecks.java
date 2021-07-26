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

// Generate a certificate for the integration (provide at least Country Name, can use defaults for rest)
// openssl req -x509 -newkey rsa:4096 -keyout /tmp/integration-key.pem -out /tmp/integration-cert.pem -days 365 -nodes

// Create a secret containing the generated certificate
// kubectl create secret tls my-tls-secret --cert=/tmp/integration-cert.pem --key=/tmp/integration-key.pem

// Run the integration
/*
kamel run \
  --property quarkus.http.ssl.certificate.file=/etc/camel/conf.d/_secrets/my-tls-secret/tls.crt \
  --property quarkus.http.ssl.certificate.key-file=/etc/camel/conf.d/_secrets/my-tls-secret/tls.key \
  --config secret:my-tls-secret \
  HttpsHealthChecks.java \
  --trait container.port=8443 \
  --trait container.probes-enabled=true \
  --trait container.liveness-scheme=HTTPS \
  --trait container.readiness-scheme=HTTPS
*/

import org.apache.camel.builder.RouteBuilder;

public class HttpsHealthChecks extends RouteBuilder {
    @Override
    public void configure() throws Exception {
        from("timer:tick")
            .setBody()
            .constant("Hello Camel K!")
            .to("log:info");
    }
}
