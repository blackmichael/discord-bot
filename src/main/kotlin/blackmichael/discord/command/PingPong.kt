package blackmichael.discord.command

import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener

class PingPong(eventListener: EnhancedEventListener, commandSet: CommandSet) : Command(eventListener, commandSet) {

    override val definition = Definition(
        name = "Ping Pong",
        description = "Replies back to your ping.",
        usage = "ping"
    )

    override fun EnhancedEventListener.run() {
        commandSet.command("ping") {
            reply("PONG")
        }
    }
}

fun Bot.pingPongCommand(cmdSet: CommandSet): Command = PingPong(this, cmdSet)
