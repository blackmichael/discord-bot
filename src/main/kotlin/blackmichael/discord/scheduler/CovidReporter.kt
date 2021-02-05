package blackmichael.discord.scheduler

import blackmichael.discord.DiscordBot
import blackmichael.discord.io.covid.CovidDataService
import java.io.Closeable
import java.time.Duration
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import org.slf4j.LoggerFactory

class CovidReporter(
    private val config: Config,
    private val bot: DiscordBot,
    private val covidDataService: CovidDataService
) : Closeable {
    data class Config(
        val channelId: String,
        val timeOfDay: String,
        val timezone: String = "America/Chicago"
    )

    companion object {
        private val logger = LoggerFactory.getLogger(this::class.java)
    }

    private val scheduler = Executors.newScheduledThreadPool(1)

    private val sendMessage = Runnable {
        runBlocking {
            launch {
                logger.info("running covid report...")
                val covidData = covidDataService.getMinnesotaCovidData()
                bot.sendMessage(config.channelId, covidData.toString())
                logger.debug("report complete!")
            }
        }
    }

    private fun getInitialDelay(): Duration {
        val zone = ZoneId.of(config.timezone)
        val now = ZonedDateTime.now(zone)
        val today = LocalDate.now()
        val targetTime = LocalTime.parse(config.timeOfDay)
        val targetDateTime = ZonedDateTime.of(LocalDateTime.of(today, targetTime), zone)
        val adjustedTargetDateTime = if (now.isAfter(targetDateTime)) {
            targetDateTime.plusDays(1)
        } else {
            targetDateTime
        }

        return Duration.between(ZonedDateTime.now(), adjustedTargetDateTime)
    }

    fun start() {
        val initialDelay = getInitialDelay().toSeconds()
        val period = Duration.ofDays(1).toSeconds()
        scheduler.scheduleAtFixedRate(sendMessage, initialDelay, period, TimeUnit.SECONDS)
    }

    override fun close() {
        scheduler.shutdown()

        try {
            scheduler.awaitTermination(15, TimeUnit.SECONDS)
        } catch (e: InterruptedException) {
            logger.warn("scheduler took longer than 15 seconds to shut down")
        }
    }
}
