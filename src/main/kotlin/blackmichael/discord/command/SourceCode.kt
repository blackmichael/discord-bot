package blackmichael.discord.command

import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener

class SourceCode(
    eventListener: EnhancedEventListener,
    commandSet: CommandSet,
    val config: Config
) : Command(eventListener, commandSet) {

    override val definition = Definition(
        name = "Source code",
        description = "View the source code for this bot.",
        usage = "show me the code"
    )

    data class Config(
        val url: String
    )

    override fun EnhancedEventListener.run() {
        commandSet.command("show me the code") {
            reply("View the source code here: ${config.url}")
        }
    }
}

fun Bot.sourceCodeCommand(commandSet: CommandSet, config: SourceCode.Config): Command = SourceCode(this, commandSet, config)