package blackmichael.discord.command

import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener

class Help(
    eventListener: EnhancedEventListener,
    commandSet: CommandSet,
    prefixes: List<String>,
    private val commands: List<Command>
) : Command(eventListener, commandSet, prefixes, "help") {

    override val definition = Definition(
        name = "Help",
        description = "Provides a list of available commands.",
        usage = "help"
    )

    override suspend fun EnhancedEventListener.action(message: Message) {
        val output = commands.joinToString(
            prefix = "Available commands:\n",
            separator = "\n",
            transform = {
                """
                    ${it.definition.name}:
                        _${it.definition.description}_
                        usage:
                            ${it.definition.usage}
                    """.trimIndent()
            }
        )

        message.reply(output)
    }
}

fun Bot.helpCommand(commandSet: CommandSet, prefixes: List<String>, commands: List<Command>): Command =
    Help(this, commandSet, prefixes, commands)
