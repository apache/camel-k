
from('knative:channel/messages')
  .split().tokenize(" ")
  .log('sending ${body} to words channel')
  .to('knative:channel/words')