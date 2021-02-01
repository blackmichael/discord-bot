package blackmichael.discord

import blackmichael.discord.command.Command
import blackmichael.discord.command.SourceCode
import blackmichael.discord.command.classifier.subjectClassifierCommand
import blackmichael.discord.command.covidCommand
import blackmichael.discord.command.echoCommand
import blackmichael.discord.command.helpCommand
import blackmichael.discord.command.pingPongCommand
import blackmichael.discord.command.sourceCodeCommand
import blackmichael.discord.command.whiteClawsCommand
import blackmichael.discord.io.covid.CovidDataService
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.commands
import java.io.Closeable
import kotlinx.coroutines.runBlocking
import org.slf4j.LoggerFactory

class DiscordBot(private val config: Config, private val covidDataService: CovidDataService) : Closeable {
    data class Config(
        val token: String,
        val prefixes: List<String>,
        val botId: String,
        val sourceCode: SourceCode.Config
    )

    private val log = LoggerFactory.getLogger(javaClass)

    private val bot = Bot(config.token).apply {
        commands("") {
            val commands: List<Command> = listOf(
                pingPongCommand(this, config.prefixes),
                subjectClassifierCommand(this, config.prefixes, config.botId),
                sourceCodeCommand(this, config.prefixes, config.sourceCode),
                whiteClawsCommand(this, config.prefixes),
                echoCommand(this, config.prefixes),
                covidCommand(this, config.prefixes, covidDataService)
            )

            commands.forEach { it.listen() }
            helpCommand(this, config.prefixes, commands).listen()
        }
    }

    suspend fun start() {
        log.info("starting discord bot")
        bot.start()
    }

    override fun close() {
        log.info("closing discord bot")
        runBlocking {
            bot.shutdown()
        }
    }
}
