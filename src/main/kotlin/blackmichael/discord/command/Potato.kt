package blackmichael.discord.command

import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener
import kotlin.random.Random

class Potato(
    eventListener: EnhancedEventListener,
    commandSet: CommandSet,
    private val botId: String
) : Command(eventListener, commandSet) {

    override val definition = Definition(
        name = "Is this a potato?",
        description = "Determines if something is a potato or not.",
        usage = "is <subject> a potato?"
    )

    override fun EnhancedEventListener.run() {
        commandSet.command("is") {
            val subject = content.removePrefix("${commandSet.prefix}is ")
                .removeSuffix("a potato?")
                .trim()
            // the bot is always a potato, obviously
            val message = if (subject.trim() == botId.trim()) {
                isPotatoMessage(subject, true)
            } else {
                isPotatoMessage(subject)
            }

            reply(message)
        }
    }

    private fun isPotatoMessage(subject: String, isPotato: Boolean = Random.nextBoolean()) =
        when (isPotato) {
            true -> "Yes, $subject is a potato."
            false -> "Nah, $subject is not a potato."
        }
}

fun Bot.potatoCommand(cmdSet: CommandSet, botId: String): Command = Potato(this, cmdSet, botId)
