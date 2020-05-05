package blackmichael.discord.server

import io.ktor.application.call
import io.ktor.http.ContentType
import io.ktor.response.respondText
import io.ktor.routing.get
import io.ktor.routing.routing
import io.ktor.server.engine.ApplicationEngine
import io.ktor.server.engine.embeddedServer
import io.ktor.server.netty.Netty
import java.io.Closeable
import org.slf4j.LoggerFactory

class SimpleServer(config: Config) : Closeable {
    data class Config(
        val port: Int
    )

    private val log = LoggerFactory.getLogger(javaClass)

    private val server: ApplicationEngine = embeddedServer(Netty, config.port) {
        routing {
            get("/") {
                call.respondText("RUNNING", ContentType.Text.Plain)
            }
        }
    }

    fun start() {
        log.info("starting server")
        server.start(wait = false)
    }

    override fun close() {
        log.info("closing server")
        server.stop(0, 0)
    }
}
