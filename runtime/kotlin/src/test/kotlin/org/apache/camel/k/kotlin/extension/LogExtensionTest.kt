package org.apache.camel.k.kotlin.extension

import org.apache.camel.component.log.LogComponent
import org.apache.camel.impl.DefaultCamelContext
import org.apache.camel.impl.DefaultExchange
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test

class LogExtensionTest {
    @Test
    @Throws(Exception::class)
    fun `invoke extension method - formatter`()  {
        val ctx = DefaultCamelContext()

        var log = LogComponent()
        log.formatter {
            e -> "body: " + e.getIn().body
        }

        var ex = DefaultExchange(ctx)
        ex.getIn().body = "hello"

        assertThat(log.exchangeFormatter.format(ex)).isEqualTo("body: hello")
    }
}