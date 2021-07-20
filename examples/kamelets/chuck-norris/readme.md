# Camel Example Kamelet Chuck Norris

## Introduction

This example shows how you can use an out of the box Kamelet with your Camel applications.

This example uses the Chuck Norris Kamelet that periodically gets a joke from the Chuck Norris internet database.

A Camel routes is _coded_ in the `chuck.xml` file using the XML DSL that uses the Kamelet,
and log the result from the Kamelet to the console.

## Running the example

Just run the integration via:
```
$ kamel run chuck.xml
```
You should be able to see the new integration running after some time:
```
$ kamel get
NAME	PHASE	KIT
chuck	Running	kit-bu9d2r22hhmoa6qrtc2g
```

You can then show the logs of the running pod to see the Chuck Norris jokes:

```
$ kamel log chuck
```

## Help and contributions

If you hit any problem using Camel or have some feedback, then please
https://camel.apache.org/community/support/[let us know].

We also love contributors, so
https://camel.apache.org/community/contributing/[get involved] :-)

The Camel riders!
