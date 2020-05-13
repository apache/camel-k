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

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.SagaPropagation;
import org.apache.camel.model.rest.RestParamType;
import org.apache.camel.service.lra.LRASagaService;

public class Payment extends RouteBuilder {
	@Override
	public void configure() throws Exception {
		LRASagaService service = new LRASagaService();
		service.setCoordinatorUrl("http://lra-coordinator");
		service.setLocalParticipantUrl("http://payment");
		getContext().addService(service);

		rest("/api/").post("/pay")
                    .param().type(RestParamType.query).name("type").required(true).endParam()
                    .param().type(RestParamType.header).name("id").required(true).endParam()
                    .route()
                    .saga()
                        .propagation(SagaPropagation.MANDATORY)
                        .option("id", header("id"))
                        .compensation("direct:cancelPayment")
                    .log("Paying ${header.type} for order #${header.id}")
                    .choice()
                        .when(x -> Math.random() >= 0.85)
                            .throwException(new RuntimeException("Random failure during payment"))
                    .end();

                from("direct:cancelPayment")
                    .log("Payment #${header.id} has been cancelled");
	}
}
