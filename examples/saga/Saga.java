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
import org.apache.camel.service.lra.LRASagaService;
import org.apache.camel.Exchange;

public class Saga extends RouteBuilder {
	@Override
	public void configure() throws Exception {
		// Enable rest binding
        rest();

		LRASagaService service = new LRASagaService();
		service.setCoordinatorUrl("http://lra-coordinator");
		service.setLocalParticipantUrl("http://saga");
		getContext().addService(service);

		from("timer:clock?period=5000")
			.saga()
			.setHeader("id", header(Exchange.TIMER_COUNTER))
			.setHeader(Exchange.HTTP_METHOD, constant("POST"))
			.log("Executing saga #${header.id}")
			.to("http://train/api/train/buy/seat?bridgeEndpoint=true")
			.to("http://flight/api/flight/buy?bridgeEndpoint=true");

	}
}
