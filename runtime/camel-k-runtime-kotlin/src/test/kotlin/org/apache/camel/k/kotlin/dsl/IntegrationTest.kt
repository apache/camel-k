package org.apache.camel.k.kotlin.dsl

import org.apache.camel.Processor
import org.apache.camel.component.log.LogComponent
import org.apache.camel.component.seda.SedaComponent
import org.apache.camel.k.Runtime
import org.apache.camel.k.jvm.ApplicationRuntime
import org.apache.camel.k.listener.RoutesConfigurer
import org.apache.camel.spi.ExchangeFormatter
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference

class IntegrationTest {
    @Test
    fun `load integration with rest`() {
        var runtime = ApplicationRuntime()
        runtime.addListener(RoutesConfigurer.forRoutes("classpath:routes-with-rest.kts"))
        runtime.addListener(Runtime.Phase.Started) { runtime.stop() }
        runtime.run()

        assertThat(runtime.context.restConfiguration.host).isEqualTo("my-host")
        assertThat(runtime.context.restConfiguration.port).isEqualTo(9192)
        assertThat(runtime.context.getRestConfiguration("undertow", false).host).isEqualTo("my-undertow-host")
        assertThat(runtime.context.getRestConfiguration("undertow", false).port).isEqualTo(9193)
        assertThat(runtime.context.restDefinitions.size).isEqualTo(1)
        assertThat(runtime.context.restDefinitions[0].path).isEqualTo("/my/path")
    }

    @Test
    fun `load integration with binding`() {
        var runtime = ApplicationRuntime()
        runtime.addListener(RoutesConfigurer.forRoutes("classpath:routes-with-bindings.kts"))
        runtime.addListener(Runtime.Phase.Started) { runtime.stop() }
        runtime.run()

        assertThat(runtime.context.registry.lookupByName("my-entry")).isEqualTo("myRegistryEntry1")
        assertThat(runtime.context.registry.lookupByName("my-proc")).isInstanceOf(Processor::class.java)
    }

    @Test
    fun `load integration with component configuration`() {
        val sedaSize = AtomicInteger()
        val sedaConsumers = AtomicInteger()
        val mySedaSize = AtomicInteger()
        val mySedaConsumers = AtomicInteger()
        val format = AtomicReference<ExchangeFormatter>()

        var runtime = ApplicationRuntime()
        runtime.addListener(RoutesConfigurer.forRoutes("classpath:routes-with-component-configuration.kts"))
        runtime.addListener(Runtime.Phase.Started) {
            val seda = runtime.context.getComponent("seda", SedaComponent::class.java)
            val mySeda = runtime.context.getComponent("mySeda", SedaComponent::class.java)
            val log = runtime.context.getComponent("log", LogComponent::class.java)

            sedaSize.set(seda!!.queueSize)
            sedaConsumers.set(seda.concurrentConsumers)
            mySedaSize.set(mySeda!!.queueSize)
            mySedaConsumers.set(mySeda.concurrentConsumers)
            format.set(log!!.exchangeFormatter)

            runtime.stop()
        }

        runtime.run()

        assertThat(sedaSize.get()).isEqualTo(1234)
        assertThat(sedaConsumers.get()).isEqualTo(12)
        assertThat(mySedaSize.get()).isEqualTo(4321)
        assertThat(mySedaConsumers.get()).isEqualTo(21)
        assertThat(format.get()).isNotNull
    }
}