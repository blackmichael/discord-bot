package blackmichael.discord.io.covid

import java.math.BigDecimal
import java.math.MathContext
import java.math.RoundingMode
import java.time.LocalDate

data class CovidData(
    val date: LocalDate,
    val source: String,
    val totalPositiveCases: Int,
    val totalDeaths: Int,
    val totalTests: Int,
    val totalHospitalizations: Int,
    val totalIcuHospitalizations: Int,
    val newPositiveCases: Int,
    val newDeaths: Int,
    val newTests: Int,
    val newHospitalizations: Int,
    val newIcuHospitalizations: Int
) {
    private fun getPercentage(dividend: Int, divisor: Int): BigDecimal =
        dividend.toBigDecimal()
            .divide(divisor.toBigDecimal(), 4, RoundingMode.UP) * BigDecimal(100, MathContext(1))

    override fun toString(): String = """
        _${date.month.name.toLowerCase().capitalize()} ${date.dayOfMonth}, ${date.year} Update_
        
        Positive cases: 
            $newPositiveCases new cases (${getPercentage(newPositiveCases, newTests)}% of new tests)
            $totalPositiveCases total
        
        Tests: 
            $newTests new tests
            $totalTests total
        
        Hospitalizations: 
            $newHospitalizations new hospitalizations ($newIcuHospitalizations in ICU)
            $totalHospitalizations total (${getPercentage(totalHospitalizations, totalPositiveCases)}% of total cases)
            $totalIcuHospitalizations ICU total (${getPercentage(totalIcuHospitalizations, totalHospitalizations)}% of total hospitalizations)
        
        Deaths:
            $newDeaths new deaths
            $totalDeaths total
    """.trimIndent()
}
