repositories {
    jcenter()
    mavenCentral()
    maven("https://plugins.gradle.org/m2/")
}

plugins {
    application
    kotlin("jvm") version "1.3.72"
    id("org.jlleitschuh.gradle.ktlint") version "9.2.1"
}

java {
    sourceCompatibility = JavaVersion.VERSION_12
    targetCompatibility = JavaVersion.VERSION_12
}

application {
    mainClassName = "blackmichael.discord.DiscordBotKt"
}

dependencies {
    implementation(kotlin("stdlib"))

    implementation("com.jessecorbett:diskord:1.6.2")
    implementation("io.github.config4k:config4k:0.4.1")

    implementation("org.slf4j:slf4j-api:1.7.30")
    implementation("ch.qos.logback:logback-classic:1.1.7")
}
