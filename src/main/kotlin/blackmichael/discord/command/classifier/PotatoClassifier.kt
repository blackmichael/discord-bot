package blackmichael.discord.command.classifier

import kotlin.random.Random

class PotatoClassifier(
    private val botId: String
) : Classifier {

    override fun classify(subject: String): Boolean {
        return if (subject.trim() == botId.trim()) {
            true
        } else {
            Random.nextBoolean()
        }
    }

    override fun getAffirmativeMessage(subject: String): String {
        return "Yes, $subject is a potato."
    }

    override fun getNegativeMessage(subject: String): String {
        return "Nah, $subject is not a potato."
    }
}
