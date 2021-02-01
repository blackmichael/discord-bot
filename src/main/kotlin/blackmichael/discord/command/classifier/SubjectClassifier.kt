package blackmichael.discord.command.classifier

import blackmichael.discord.command.Command
import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener
import kotlin.random.Random

class SubjectClassifier(
    eventListener: EnhancedEventListener,
    commandSet: CommandSet,
    prefixes: List<String>,
    private val botId: String
) : Command(eventListener, commandSet, prefixes, "is") {

    private val snarkyResponses = listOf(
        "You cannot do that yet.",
        "I'm trying my best, but my creators failed me",
        "Maybe someday I'll know the answer. Today is not that day"
    )

    override val definition = Definition(
        name = "Is this a thing?",
        description = "Determines if something is, or is not.",
        usage = "is <subject> a (potato|hobby)?"
    )

    override suspend fun EnhancedEventListener.action(message: Message) {
        val content = message.content.indexOf("is ").let { message.content.substring(it + 3) }
        val isPotatoClassifier = content.contains("potato")
        val isHobbyClassifier = content.contains("hobby")
        when {
            isPotatoClassifier -> {
                val subject = content
                    .removeSuffix("a potato?")
                    .trim()
                message.reply(PotatoClassifier(botId).getMessage(subject))
            }
            isHobbyClassifier -> {
                val subject = content
                    .removeSuffix("a hobby?")
                    .trim()
                message.reply(HobbyClassifier().getMessage(subject))
            }
            else -> {
                val snarkyChoice = Random.nextInt(snarkyResponses.size)
                message.reply(snarkyResponses[snarkyChoice])
            }
        }
    }
}

fun Bot.subjectClassifierCommand(cmdSet: CommandSet, prefixes: List<String>, botId: String): SubjectClassifier =
    SubjectClassifier(this, cmdSet, prefixes, botId)
