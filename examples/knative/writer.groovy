from('timer:messages?period=10s')
  .setBody().constant('the-body')
  .to('knative:endpoint/reader')