/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;

public class NetworkPolicyConsumer extends RouteBuilder {
    @Override
    public void configure() {
        onException(Exception.class)
            .maximumRedeliveries(0)
            .handled(true)
            .log("ACCESS_DENIED");

        from("timer:network-policy?delay=1000&period=1000&repeatCount=30")
            .to("http://{{serviceName}}/hello?throwExceptionOnFailure=false&connectTimeout=3000&responseTimeout=3000")
            .choice()
                .when(header(Exchange.HTTP_RESPONSE_CODE).isEqualTo(200))
                    .log("ACCESS_GRANTED")
                .otherwise()
                    .log("ACCESS_DENIED");
    }
}
