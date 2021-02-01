package blackmichael.discord.command

import blackmichael.discord.io.covid.CovidDataService
import com.jessecorbett.diskord.api.model.Message
import com.jessecorbett.diskord.dsl.Bot
import com.jessecorbett.diskord.dsl.CommandSet
import com.jessecorbett.diskord.util.EnhancedEventListener
import org.slf4j.LoggerFactory

class Covid(
    eventListener: EnhancedEventListener,
    commandSet: CommandSet,
    prefixes: List<String>,
    private val covidDataService: CovidDataService
) : Command(eventListener, commandSet, prefixes, "COVID") {

    override val definition = Definition(
        name = "COVID Update",
        description = "Fetches the latest COVID data",
        usage = "COVID"
    )

    companion object {
        private val logger = LoggerFactory.getLogger(this::class.java)
    }

    override suspend fun EnhancedEventListener.action(message: Message) {
        val response = try {
            covidDataService.getMinnesotaCovidData().toString()
        } catch (e: Exception) {
            e.printStackTrace()
            logger.error("unable to fetch COVID data: ${e.message}")
            "Unable to fetch COVID data"
        }

        message.reply(response)
    }
}

fun Bot.covidCommand(cmdSet: CommandSet, prefixes: List<String>, covidDataService: CovidDataService): Command =
    Covid(this, cmdSet, prefixes, covidDataService)
