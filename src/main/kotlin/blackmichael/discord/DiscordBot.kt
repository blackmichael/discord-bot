package blackmichael.discord

import blackmichael.discord.command.Command
import blackmichael.discord.command.SourceCode
import blackmichael.discord.command.classifier.SubjectClassifier
import blackmichael.discord.command.classifier.subjectClassifierCommand
import blackmichael.discord.command.helpCommand
import blackmichael.discord.command.pingPongCommand
import blackmichael.discord.command.sourceCodeCommand
import com.jessecorbett.diskord.dsl.bot
import com.jessecorbett.diskord.dsl.commands
import com.typesafe.config.ConfigFactory
import io.github.config4k.extract

suspend fun main() {
    val config = ConfigFactory.load()
    val bot = DiscordBot(config.extract("bot"))
    bot.start()
}

class DiscordBot(val config: Config) {
    data class Config(
        val token: String,
        val prefix: String,
        val botId: String,
        val sourceCode: SourceCode.Config
    )

    suspend fun start() {
        bot(config.token) {
            commands(config.prefix) {
                val commands: List<Command> = listOf(
                    pingPongCommand(this),
                    subjectClassifierCommand(this, config.botId),
                    sourceCodeCommand(this, config.sourceCode)
                )

                commands.forEach { it.listen() }
                helpCommand(this, commands).listen()
            }
        }
    }
}
