package blackmichael.discord

import blackmichael.discord.server.SimpleServer
import com.typesafe.config.ConfigFactory
import io.github.config4k.extract
import kotlin.system.exitProcess

suspend fun main() {
    val config = ConfigFactory.load()
    val bot = DiscordBot(config.extract("bot"))
    val server = SimpleServer(config.extract("server"))

    Runtime.getRuntime().addShutdownHook(Thread {
        server.close()
        exitProcess(1)
    })

    server.start()
    bot.start()
}
