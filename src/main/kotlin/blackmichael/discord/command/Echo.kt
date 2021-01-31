package blackmichael.discord.command

import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener

class Echo(eventListener: EnhancedEventListener, commandSet: CommandSet, prefixes: List<String>) :
    Command(eventListener, commandSet, prefixes, "echo") {
    override val definition = Definition(
        name = "Echo",
        description = "Echos back what was sent.",
        usage = "echo <anything>"
    )

    override suspend fun EnhancedEventListener.action(message: Message) {
        val content = when (val rawContent = message.content.split("echo")[1]) {
            "" -> " "
            else -> rawContent.trim()
        }
        message.reply("```$content```")
    }
}

fun Bot.echoCommand(commandSet: CommandSet, prefixes: List<String>): Command = Echo(this, commandSet, prefixes)
