package blackmichael.discord.command

import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener

class SourceCode(
    eventListener: EnhancedEventListener,
    commandSet: CommandSet,
    prefixes: List<String>,
    private val config: Config
) : Command(eventListener, commandSet, prefixes, "show me the code") {

    override val definition = Definition(
        name = "Source code",
        description = "View the source code for this bot.",
        usage = "show me the code"
    )

    data class Config(
        val url: String
    )

    override suspend fun EnhancedEventListener.action(message: Message) {
        message.reply("View the source code here: ${config.url}")
    }
}

fun Bot.sourceCodeCommand(commandSet: CommandSet, prefixes: List<String>, config: SourceCode.Config): Command =
    SourceCode(this, commandSet, prefixes, config)
