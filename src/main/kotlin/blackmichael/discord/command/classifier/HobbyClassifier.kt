package blackmichael.discord.command.classifier

import kotlin.random.Random

class HobbyClassifier : Classifier {

    override fun classify(subject: String): Boolean {
         return Random.nextBoolean()
    }

    override fun getAffirmativeMessage(subject: String): String {
        return "Yes, $subject is a hobby."
    }

    override fun getNegativeMessage(subject: String): String {
        return "Nah, $subject is not a hobby."
    }
}