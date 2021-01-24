package blackmichael.discord.command

import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener
import kotlin.random.Random
import kotlin.text.Regex

class WhiteClaws(eventListener: EnhancedEventListener, commandSet: CommandSet) : Command(eventListener, commandSet) {

    override val definition = Definition(
        name = "White Claws",
        description = "Tells you how many whiteclaws someone will drink.",
        usage = "how many whiteclaws (am I|is <subject>) (going to|gonna) drink tonight?"
    )

    val commandRegex = Regex("how many whiteclaws (am I|is .*) (going to|gonna) drink( tonight)?\\?$")

    override fun EnhancedEventListener.run() {
        commandSet.command("how many whiteclaws") {
            val match = commandRegex.find(content)
            match?.let {
                val (subject) = it.destructured
                val messageSubject = when (subject) {
                    "am I" -> "You're"
                    else -> "${subject.removePrefix("is ")} is"
                }
                val numOfDrinks = Random.nextInt(100)
                val messageComments = if (numOfDrinks == 0) " Those are rookie numbers." else ""

                reply("$messageSubject gonna drink $numOfDrinks whiteclaws tonight.$messageComments")
            }
        }
    }
}

fun Bot.whiteClawsCommand(cmdSet: CommandSet): Command = WhiteClaws(this, cmdSet)
