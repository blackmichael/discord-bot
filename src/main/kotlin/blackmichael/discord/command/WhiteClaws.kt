package blackmichael.discord.command

import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener
import kotlin.random.Random

class WhiteClaws(eventListener: EnhancedEventListener, commandSet: CommandSet, prefixes: List<String>) :
    Command(eventListener, commandSet, prefixes, "how many whiteclaws") {

    override val definition = Definition(
        name = "White Claws",
        description = "Tells you how many whiteclaws someone will drink.",
        usage = "how many whiteclaws (am I|is <subject>) (going to|gonna) drink tonight?"
    )

    private val commandRegex = Regex("how many whiteclaws (am I|is .*) (going to|gonna) drink( tonight)?\\?$")

    override suspend fun EnhancedEventListener.action(message: Message) {
        val match = commandRegex.find(message.content)
        match?.let {
            val (subject) = it.destructured
            val messageSubject = when (subject) {
                "am I" -> "You're"
                else -> "${subject.removePrefix("is ")} is"
            }
            val numOfDrinks = Random.nextInt(100)
            val messageComments = if (numOfDrinks == 0) " Those are rookie numbers." else ""

            message.reply("$messageSubject gonna drink $numOfDrinks whiteclaws tonight.$messageComments")
        }
    }
}

fun Bot.whiteClawsCommand(cmdSet: CommandSet, prefixes: List<String>): Command =
    WhiteClaws(this, cmdSet, prefixes)
