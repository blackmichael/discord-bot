package blackmichael.discord.command

import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener

class PingPong(eventListener: EnhancedEventListener, commandSet: CommandSet, prefixes: List<String>) :
    Command(eventListener, commandSet, prefixes, "ping") {

    override val definition = Definition(
        name = "Ping Pong",
        description = "Replies back to your ping.",
        usage = "ping"
    )

    override suspend fun EnhancedEventListener.action(message: Message) {
        message.reply("PONG")
    }
}

fun Bot.pingPongCommand(cmdSet: CommandSet, prefixes: List<String>): Command = PingPong(this, cmdSet, prefixes)
