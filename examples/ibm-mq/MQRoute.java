// camel-k: language=java
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

import com.ibm.mq.jms.MQQueueConnectionFactory;
import com.ibm.msg.client.wmq.WMQConstants;
import org.apache.camel.CamelContext;
import org.apache.camel.ProducerTemplate;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.jms.JmsComponent;
import org.apache.camel.impl.DefaultCamelContext;

public class MQRoute extends RouteBuilder {

    static String mqHost = "10.108.69.161";
    static int mqPort = 1414;
    static String mqQueueManager = "QM1";
    static String mqChannel = "DEV.APP.SVRCONN";
    static String mqQueue = "DEV.QUEUE.1";
    static String user = "app";
    static String password = "ibmmqpasswd";

    @Override
    public void configure() {
        MQQueueConnectionFactory mqFactory = createWMQConnectionFactory(mqHost);
        getContext().getRegistry().bind("mqConnectionFactory", mqFactory);
        
        from("timer:tick")
            .setBody()
              .simple("Hello Camel K! #${exchangeProperty.CamelTimerCounter}")
            .to("jms:queue:" + mqQueue + "?connectionFactory=#mqConnectionFactory");

        from("jms:queue:" + mqQueue + "?connectionFactory=#mqConnectionFactory")
            .to("log:info");
    }

    private MQQueueConnectionFactory createWMQConnectionFactory(String mqHost) {
        MQQueueConnectionFactory mqQueueConnectionFactory = new MQQueueConnectionFactory();
        try {
            mqQueueConnectionFactory.setHostName(mqHost);
            mqQueueConnectionFactory.setChannel(mqChannel);
            mqQueueConnectionFactory.setPort(mqPort);
            mqQueueConnectionFactory.setQueueManager(mqQueueManager);
            mqQueueConnectionFactory.setTransportType(WMQConstants.WMQ_CM_CLIENT);
            mqQueueConnectionFactory.setStringProperty(WMQConstants.USERID, user);
            mqQueueConnectionFactory.setStringProperty(WMQConstants.PASSWORD, password);
        } catch (Exception e) {
            e.printStackTrace();
        }
        return mqQueueConnectionFactory;
    }
}
