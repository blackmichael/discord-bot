package blackmichael.discord.command.classifier

import blackmichael.discord.command.Command
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener
import kotlin.random.Random

class SubjectClassifier(
        eventListener: EnhancedEventListener,
        commandSet: CommandSet,
        private val botId: String
) : Command(eventListener, commandSet) {

    val snarkyResponses = listOf<String>(
            "You cannot do that yet.",
            "I'm trying my best, but my creators failed me",
            "Maybe someday I'll know the answer. Today is not that day")

    override val definition = Definition(
            name = "Is this a potato?",
            description = "Determines if something is, or is not.",
            usage = "is <subject> a (potato|hobby)?"
    )

    override fun EnhancedEventListener.run() {
        commandSet.command("is") {
            val isPotatoClassifier = content.removePrefix("${commandSet.prefix}is ").contains("potato")
            val isHobbyClassifier = content.removePrefix("${commandSet.prefix}is ").contains("potato")
            if (isPotatoClassifier) {
                val subject = content.removePrefix("${commandSet.prefix}is ")
                        .removeSuffix("a potato?")
                        .trim()
                reply(PotatoClassifier(botId).getMessage(subject))
            } else if (isHobbyClassifier) {
                val subject = content.removePrefix("${commandSet.prefix}is ")
                        .removeSuffix("a hobby?")
                        .trim()
                reply(HobbyClassifier().getMessage(subject))
            } else {
                val snarkyChoice = Random.nextInt(snarkyResponses.size)
                reply(snarkyResponses[snarkyChoice])
            }
        }
    }
}

fun Bot.subjectClassifierCommand(cmdSet: CommandSet, botId: String): SubjectClassifier = SubjectClassifier(this, cmdSet, botId)
