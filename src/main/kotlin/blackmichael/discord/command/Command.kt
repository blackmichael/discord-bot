package blackmichael.discord.command

import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.util.EnhancedEventListener

abstract class Command(
    private val eventListener: EnhancedEventListener,
    protected val commandSet: CommandSet,
    private val prefixes: List<String>,
    private val commandText: String
) {
    data class Definition(
        val name: String,
        val description: String,
        val usage: String
    )

    abstract val definition: Definition

    abstract suspend fun EnhancedEventListener.action(message: Message)

    fun listen() =
        eventListener.apply {
            prefixes.forEach { prefix ->
                commandSet.command("$prefix$commandText") {
                    action(this)
                }
            }
        }
}
