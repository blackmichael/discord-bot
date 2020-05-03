package blackmichael.discord

import com.jessecorbett.diskord.dsl.bot
import com.jessecorbett.diskord.dsl.command
import com.jessecorbett.diskord.dsl.commands

private val token = "<token>"

suspend fun main() {
    bot(token) {
        commands {
            command("ping") {
                reply("PONG")
            }
        }
    }
}