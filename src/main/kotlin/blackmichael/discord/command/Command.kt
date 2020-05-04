package blackmichael.discord.command

import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener

abstract class Command(val eventListener: EnhancedEventListener, protected val commandSet: CommandSet) {
    data class Definition(
        val name: String,
        val description: String,
        val usage: String
    )

    abstract val definition: Definition

    abstract fun EnhancedEventListener.run()

    fun listen() = eventListener.run()
}
