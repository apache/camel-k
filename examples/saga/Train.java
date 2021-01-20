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
import org.apache.camel.Exchange;

public class Train extends RouteBuilder {
	@Override
	public void configure() throws Exception {
		LRASagaService service = new LRASagaService();
		service.setCoordinatorUrl("http://lra-coordinator");
		service.setLocalParticipantUrl("http://train");
		getContext().addService(service);

		rest("/api/").post("/train/buy/seat")
                    .param().type(RestParamType.header).name("id").required(true).endParam()
                    .route()
                    .saga()
                        .propagation(SagaPropagation.SUPPORTS)
                        .option("id", header("id"))
                        .compensation("direct:cancelPurchase")
                    .log("Buying train seat #${header.id}")
                    .removeHeaders("CamelHttp.*")
                    .to("http://payment/api/pay?httpMethod=POST&type=train")
                    .log("Payment for train #${header.id} done");

               from("direct:cancelPurchase")
                    .log("Train purchase #${header.id} has been cancelled");
	}
}
