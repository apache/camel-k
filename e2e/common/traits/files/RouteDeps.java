import java.lang.Exception;
import java.lang.Override;
import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;

public class RouteDeps extends RouteBuilder {

		String s = "random";

    @Override
    public void configure() throws Exception {
        String params = "random_string";
        from("telegram:" + params)
            .setHeader("test",simple("${in.header.fileName}"))
            .pollEnrich()
                .simple("aws2-s3://data?fileName=${in.header.fileName}&deleteAfterRead=false")
                .unmarshal().jacksonXml()
                .to("mongodb:test")
            .end()
            .unmarshal().zipFile()
            .to("dropbox:random")
            .toD("caffeine-cache:" + s)
            .to("kafka:test")
            ;
    }
}

