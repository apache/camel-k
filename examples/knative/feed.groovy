from('timer:clock?period=3s')
	.setBody().constant("Hello World from Camel K")
	.to('knative:channel/messages')
	.log('sent message to messages channel')