package blackmichael.discord

import blackmichael.discord.io.covid.CovidDataService
import blackmichael.discord.server.SimpleServer
import com.typesafe.config.ConfigFactory
import io.github.config4k.extract
import kotlin.system.exitProcess

suspend fun main() {
    val config = ConfigFactory.load()
    val covidDataService = CovidDataService(config.extract("covidService"))
    val bot = DiscordBot(config.extract("bot"), covidDataService)
    val server = SimpleServer(config.extract("server"))

    suspend fun start() {
        server.start()
        bot.start()
    }

    fun close() {
        bot.close()
        covidDataService.close()
        server.close()
    }

    try {
        Runtime.getRuntime().addShutdownHook(Thread {
            close()
            exitProcess(0)
        })

        start()
    } catch (e: Exception) {
        println(e)
        exitProcess(1)
    }
}
