//
// To run this integrations use:
//
//     kamel run --runtime kotlin runtime/examples/kotlin-routes.kts
//
// Or leveraging runtime detection
//
//     kamel run runtime/examples/kotlin-routes.kts
//

val rnd = java.util.Random()

from("timer:kotlin?period=1s")
    .routeId("kotlin")
    .setBody()
        .constant("Hello Camel K!")
    .process().message {
        it.headers["RandomValue"] = rnd.nextInt()
    }
    .to("log:info?showAll=true&multiline=true")