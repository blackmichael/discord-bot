package blackmichael.discord.command.classifier

interface Classifier {

    fun getMessage(subject: String): String {
        return when (classify(subject)) {
            true -> getAffirmativeMessage(subject)
            false -> getNegativeMessage(subject)
        }
    }

    abstract fun classify(subject: String): Boolean
    abstract fun getAffirmativeMessage(subject: String): String
    abstract fun getNegativeMessage(subject: String): String
}
