// camel-k: language=groovy
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

//
// To run this integration use:
//
//     kamel run groovy examples/camel-caffeine.groovy
//

import com.github.benmanes.caffeine.cache.Caffeine

beans {
    caffeineCache = Caffeine.newBuilder().recordStats().build()
}

from('timer:tick')
  .setBody().constant('Hello')
  .process {
    it.in.headers['CamelCaffeineAction'] = 'PUT'
    it.in.headers['CamelCaffeineKey'] = 1
  }
  .toF('caffeine-cache://%s?cache=#caffeineCache', 'test')
  .log('Result of Action ${header.CamelCaffeineAction} with key ${header.CamelCaffeineKey} is: ${body}')
  .setBody().constant(null)
  .process {
    it.in.headers['CamelCaffeineAction'] = 'GET'
    it.in.headers['CamelCaffeineKey'] = 1
  }
  .toF('caffeine-cache://%s?cache=#caffeineCache', 'test')
  .log('Result of Action ${header.CamelCaffeineAction} with key ${header.CamelCaffeineKey} is: ${body}')
  .setBody().constant(null)
  .process {
    it.in.headers['CamelCaffeineAction'] = 'INVALIDATE'
    it.in.headers['CamelCaffeineKey'] = 1
  }
  .toF('caffeine-cache://%s?cache=#caffeineCache', 'test')
  .log('Invalidating entry with key ${header.CamelCaffeineKey}')
  .setBody().constant(null)
  .process {
    it.in.headers['CamelCaffeineAction'] = 'GET'
    it.in.headers['CamelCaffeineKey'] = 1
  }
  .toF('caffeine-cache://%s?cache=#caffeineCache', 'test')
  .log('The Action ${header.CamelCaffeineAction} with key ${header.CamelCaffeineKey} has result? ${header.CamelCaffeineActionHasResult}');
