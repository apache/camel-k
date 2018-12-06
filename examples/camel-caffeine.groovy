//
// To run this integrations use:
//
//     kamel run --runtime groovy examples/camel-caffeine.groovy
//

import com.github.benmanes.caffeine.cache.Caffeine

context {
    registry {
        caffeineCache = Caffeine.newBuilder().recordStats().build()
    }
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
