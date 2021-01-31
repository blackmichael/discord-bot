import com.github.jengelman.gradle.plugins.shadow.tasks.ShadowJar

repositories {
    jcenter()
    mavenCentral()
    maven("https://plugins.gradle.org/m2/")
}

plugins {
    application
    kotlin("jvm") version "1.3.72"
    id("org.jlleitschuh.gradle.ktlint") version "9.2.1"
    id("com.github.johnrengelman.shadow") version "5.2.0"
}

val build: DefaultTask by tasks
val shadowJar: ShadowJar by tasks

build.dependsOn(shadowJar)
shadowJar.apply {
    archiveName = "discord-bot.jar"
}

java {
    sourceCompatibility = JavaVersion.VERSION_12
    targetCompatibility = JavaVersion.VERSION_12
}

application {
    mainClassName = "blackmichael.discord.MainKt"
}

dependencies {
    implementation(kotlin("stdlib"))

    implementation("io.ktor:ktor-server-netty:1.3.2")
    implementation("com.jessecorbett:diskord:1.7.3")
    implementation("io.github.config4k:config4k:0.4.1")

    implementation("org.slf4j:slf4j-api:1.7.30")
    implementation("ch.qos.logback:logback-classic:1.1.7")
}
