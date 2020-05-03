plugins {
  application
  kotlin("jvm") version "1.3.72"
}

repositories {
  jcenter()
  mavenCentral()
}

apply(plugin = "org.jlleitschuh.gradle.ktlint")

java.sourceCompatibility = JavaVersion.VERSION_11
java.targetCompatibility = JavaVersion.VERSION_11

application {
  mainClassName = "blackmichael.discord.ApplicationKt"
}

dependencies {
  implementation(kotlin("stdlib"))

  implementation("com.jessecorbett:diskord:1.6.2")
  implementation("org.slf4j:slf4j-api:1.7.30")
}


