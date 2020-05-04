package blackmichael.discord.command

import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener

class Help(
    eventListener: EnhancedEventListener,
    commandSet: CommandSet,
    val commands: List<Command>
) : Command(eventListener, commandSet) {

    override val definition = Definition(
        name = "Help",
        description = "Provides a list of available commands.",
        usage = "help"
    )

    override fun EnhancedEventListener.run() {
        commandSet.command("help") {
            val message = commands.joinToString(
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

            reply(message)
        }
    }
}

fun Bot.helpCommand(commandSet: CommandSet, commands: List<Command>): Command = Help(this, commandSet, commands)