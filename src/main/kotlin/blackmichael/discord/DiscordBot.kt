package blackmichael.discord

import blackmichael.discord.command.Command
import blackmichael.discord.command.SourceCode
import blackmichael.discord.command.classifier.subjectClassifierCommand
import blackmichael.discord.command.echoCommand
import blackmichael.discord.command.helpCommand
import blackmichael.discord.command.pingPongCommand
import blackmichael.discord.command.sourceCodeCommand
import blackmichael.discord.command.whiteClawsCommand
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.commands
import kotlinx.coroutines.runBlocking
import org.slf4j.LoggerFactory
import java.io.Closeable

class DiscordBot(val config: Config) : Closeable {
    data class Config(
        val token: String,
        val prefix: String,
        val botId: String,
        val sourceCode: SourceCode.Config
    )

    private val log = LoggerFactory.getLogger(javaClass)

    private val bot = Bot(config.token).apply {
        commands(config.prefix) {
            val commands: List<Command> = listOf(
                pingPongCommand(this),
                subjectClassifierCommand(this, config.botId),
                sourceCodeCommand(this, config.sourceCode),
                whiteClawsCommand(this),
                echoCommand(this)
            )

            commands.forEach { it.listen() }
            helpCommand(this, commands).listen()
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
