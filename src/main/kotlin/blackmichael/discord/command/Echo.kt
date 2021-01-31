package blackmichael.discord.command

import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener

class Echo(eventListener: EnhancedEventListener, commandSet: CommandSet) : Command(eventListener, commandSet) {
    override val definition = Definition(
        name = "Echo",
        description = "Echos back what was sent.",
        usage = "echo <anything>"
    )

    override fun EnhancedEventListener.run() {
        commandSet.command("echo") {
            val content = when (val rawContent = this.content.split("echo")[1]) {
                "" -> " "
                else -> rawContent.trim()
            }
            reply("```$content```")
        }
    }
}

fun Bot.echoCommand(commandSet: CommandSet): Command = Echo(this, commandSet)