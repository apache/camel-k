package org.apache.camel.k.kotlin.dsl

import org.apache.camel.Processor
import org.apache.camel.component.log.LogComponent
import org.apache.camel.component.seda.SedaComponent
import org.apache.camel.main.MainListenerSupport
import org.apache.camel.main.MainSupport
import org.apache.camel.spi.ExchangeFormatter
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference

class IntegrationTest {
    @Test
    @Throws(Exception::class)
    fun `load integration with rest`() {
        var runtime = org.apache.camel.k.jvm.Runtime()
        runtime.duration = 5
        runtime.load("classpath:routes-with-rest.kts", null)
        runtime.addMainListener(object: MainListenerSupport() {
            override fun afterStart(main: MainSupport) {
                main.stop()
            }
        })

        runtime.run()

        assertThat(runtime.camelContext.restConfiguration.host).isEqualTo("my-host")
        assertThat(runtime.camelContext.restConfiguration.port).isEqualTo(9192)
        assertThat(runtime.camelContext.getRestConfiguration("undertow", false).host).isEqualTo("my-undertow-host")
        assertThat(runtime.camelContext.getRestConfiguration("undertow", false).port).isEqualTo(9193)
        assertThat(runtime.camelContext.restDefinitions.size).isEqualTo(1)
        assertThat(runtime.camelContext.restDefinitions[0].path).isEqualTo("/my/path")
    }

    @Test
    @Throws(Exception::class)
    fun `load integration with binding`() {
        var runtime = org.apache.camel.k.jvm.Runtime()
        runtime.duration = 5
        runtime.load("classpath:routes-with-bindings.kts", null)
        runtime.addMainListener(object: MainListenerSupport() {
            override fun afterStart(main: MainSupport) {
                main.stop()
            }
        })

        runtime.run()

        assertThat(runtime.camelContext.registry.lookup("my-entry")).isEqualTo("myRegistryEntry1")
        assertThat(runtime.camelContext.registry.lookup("my-proc")).isInstanceOf(Processor::class.java)
    }

    @Test
    @Throws(Exception::class)
    fun `load integration with component configuration`() {
        val sedaSize = AtomicInteger()
        val sedaConsumers = AtomicInteger()
        val mySedaSize = AtomicInteger()
        val mySedaConsumers = AtomicInteger()
        val format = AtomicReference<ExchangeFormatter>()

        var runtime = org.apache.camel.k.jvm.Runtime()
        runtime.duration = 5
        runtime.load("classpath:routes-with-component-configuration.kts", null)
        runtime.addMainListener(object : MainListenerSupport() {
            override fun afterStart(main: MainSupport) {
                val seda = runtime.camelContext.getComponent("seda", SedaComponent::class.java)
                val mySeda = runtime.camelContext.getComponent("mySeda", SedaComponent::class.java)
                val log = runtime.camelContext.getComponent("log", LogComponent::class.java)

                sedaSize.set(seda!!.queueSize)
                sedaConsumers.set(seda!!.concurrentConsumers)
                mySedaSize.set(mySeda!!.queueSize)
                mySedaConsumers.set(mySeda!!.concurrentConsumers)
                format.set(log!!.exchangeFormatter)

                main.stop()
            }
        })

        runtime.run()

        assertThat(sedaSize.get()).isEqualTo(1234)
        assertThat(sedaConsumers.get()).isEqualTo(12)
        assertThat(mySedaSize.get()).isEqualTo(4321)
        assertThat(mySedaConsumers.get()).isEqualTo(21)
        assertThat(format.get()).isNotNull()
    }
}