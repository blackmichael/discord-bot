package blackmichael.discord.io.covid

import io.ktor.client.HttpClient
import io.ktor.client.engine.apache.Apache
import io.ktor.client.request.get
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.readText
import io.ktor.http.HttpStatusCode
import java.io.Closeable
import java.io.IOException
import java.time.LocalDate
import org.jsoup.Jsoup
import org.jsoup.nodes.Element

class CovidDataService(private val config: Config) : Closeable {
    data class Config(
        val url: String
    )

    companion object {
        private val client = HttpClient(Apache) {
            expectSuccess = false
        }
    }

    suspend fun getMinnesotaCovidData(): CovidData {
        val response = client.get<HttpResponse>(config.url)
        return when (response.status) {
            HttpStatusCode.OK -> parseHtmlPage(response.readText())

            else -> throw ResourceFetchException("${response.status} ${response.readText()}")
        }
    }

    private fun parseHtmlPage(rawPage: String): CovidData {
        fun Element.parseText(): Int? =
            this.text()
            .replace(",", "")
            .toIntOrNull()

        val document = Jsoup.parse(rawPage)

        val newCases = document.getElementById("dailycasetotal")
            ?.children()
            ?.first()
            ?.children()
            ?.first()
            ?.select("td")
            ?.first()
            ?.parseText()

        val totalCases = document.getElementById("casetotal")
            ?.children()
            ?.first()
            ?.children()
            ?.first()
            ?.select("td")
            ?.first()
            ?.parseText()

        val newDeaths = document.getElementById("dailydeathtotal")
            ?.children()
            ?.first()
            ?.children()
            ?.first()
            ?.select("td")
            ?.first()
            ?.parseText()

        val deathTotal = document.getElementById("deathtotal")
            ?.children()
            ?.first()
            ?.children()
            ?.first()
            ?.select("td")
            ?.first()
            ?.parseText()

        val newTests = document.getElementById("labtable")
            ?.children()
            ?.first()
            ?.children()
            ?.last()
            ?.children()
            ?.let {
                // columns: date reported, pcr mdh lab, pcr ext lab, cumulative pcr tests, anitgen ext lab, antigen cumulative, total cumulative
                when (val dailyTotal = (it[1].parseText() ?: 0) + (it[2].parseText() ?: 0) + (it[4].parseText() ?: 0)) {
                    0 -> -1 // -1 to denote something went wrong
                    else -> dailyTotal
                }
            }

        val totalTests = document.getElementById("testtotal")
            ?.children()
            ?.first()
            ?.children()
            ?.first()
            ?.select("td")
            ?.first()
            ?.parseText()

        val hospitalizationsTotal = document.getElementById("hosptotal")
            ?.children()
            ?.first()
            ?.children()
            ?.first()
            ?.select("td")
            ?.first()
            ?.parseText()

        val icuHospitalizationsTotal = document.getElementById("hosptotal")
            ?.children()
            ?.first()
            ?.children()
            ?.last()
            ?.select("td")
            ?.first()
            ?.parseText()

        val (newIcuHospitalizations, newHospitalizations) = document.getElementById("hosptable")
            ?.children()
            ?.first()
            ?.children()
            ?.let { it[it.size - 2] } // get second to last row (last row is unknown/missing cases)
            ?.children()
            ?.let {
                it[1]?.parseText() to it[2]?.parseText()
            }
            ?: (-1 to -1) // in case element isn't found, required for destructuring

        return CovidData(
            date = LocalDate.now(),
            source = config.url,
            totalPositiveCases = totalCases ?: -1,
            totalDeaths = deathTotal ?: -1,
            totalHospitalizations = hospitalizationsTotal ?: -1,
            totalIcuHospitalizations = icuHospitalizationsTotal ?: -1,
            totalTests = totalTests ?: -1,
            newDeaths = newDeaths ?: -1,
            newHospitalizations = newHospitalizations ?: -1,
            newIcuHospitalizations = newIcuHospitalizations ?: -1,
            newPositiveCases = newCases ?: -1,
            newTests = newTests ?: -1
        )
    }

    override fun close() {
        client.close()
    }
}

class ResourceFetchException(override val message: String) : IOException()
