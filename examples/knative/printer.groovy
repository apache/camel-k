
from('knative:channel/words')
  .convertBodyTo(String.class)
  .to('log:info')
