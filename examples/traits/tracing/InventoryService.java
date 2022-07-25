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


/* 

kamel run InventoryService.java --name inventory -d camel:opentracing -d mvn:io.jaegertracing:jaeger-client:1.2.0 -d rest-api -d camel:jackson --property-file application.properties

*/

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.rest.RestBindingMode;
import org.apache.camel.component.jackson.JacksonDataFormat;
import java.text.SimpleDateFormat;
import org.apache.camel.Exchange;
import java.util.Date;
import java.util.Map;


public class InventoryService extends RouteBuilder {

    
    @Override
    public void configure() throws Exception {
        
        rest()
            .post("/notify/order")
                .to("direct:notify");

        
        JacksonDataFormat jacksonDataFormat = new JacksonDataFormat();
        jacksonDataFormat.setUnmarshalType(Map.class);
        JacksonDataFormat invDataFormat = new JacksonDataFormat();
        invDataFormat.setUnmarshalType(InventoryNotification.class);

        
        from("direct:notify")
            .setHeader(Exchange.CONTENT_TYPE, constant("application/json"))
            .unmarshal(jacksonDataFormat)
            .log("Inventory Notified ${body}")
            .bean(InventoryNotification.class, "getInventoryNotification(${body['orderId']},${body['itemId']},${body['quantity']} )")
            .marshal(invDataFormat)
            .convertBodyTo(String.class)
        ;

    }


    private static class InventoryNotification {
        private Integer orderId;
        private Integer itemId;
        private Integer quantity;
        private String department;
        private Date datetime;

        public static InventoryNotification getInventoryNotification(Integer orderId, Integer itemId, Integer quantity ){
            InventoryNotification invenNotification  = new InventoryNotification();
            invenNotification.setOrderId(orderId);
            invenNotification.setItemId(itemId);
            invenNotification.setQuantity(quantity);
            invenNotification.setDepartment("inventory");
            SimpleDateFormat formatter= new SimpleDateFormat("yyyy-MM-dd 'at' HH:mm:ss z");
            invenNotification.setDatetime(new Date(System.currentTimeMillis()));
            return invenNotification;
        }


        public void setOrderId(Integer orderId){
            this.orderId=orderId;
        }
        public void setItemId(Integer itemId){
            this.itemId=itemId;
        }
        public void setQuantity(Integer quantity){
            this.quantity=quantity;
        }
        public Integer getOrderId(){
            return this.orderId;
        }
        public Integer getItemId(){
            return this.itemId;
        }
        public Integer getQuantity(){
            return this.quantity;
        }
        public String getDepartment() {
            return department;
        }
        public void setDepartment(String department) {
            this.department = department;
        }
        public Date getDatetime() {
            return datetime;
        }
    
        public void setDatetime(Date datetime) {
            this.datetime = datetime;
        }
    }
    
}
