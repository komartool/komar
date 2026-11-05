import net.ltgt.gradle.errorprone.CheckSeverity
import net.ltgt.gradle.errorprone.errorprone

plugins {
    java
    checkstyle
    id("net.ltgt.errorprone") version "5.2.0"
}

repositories {
    mavenCentral()
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
    }
}

dependencies {
    errorprone("com.google.errorprone:error_prone_core:2.52.0")
    testImplementation(platform("org.junit:junit-bom:6.1.3"))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
    testImplementation("com.tngtech.archunit:archunit-junit5:1.5.1")
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 8
    options.compilerArgs.addAll(listOf("-Xlint:all,-options", "-Werror"))
    options.errorprone {
        disableWarningsInGeneratedCode = true
        error("MissingOverride", "UnusedVariable", "UnusedMethod", "ReferenceEquality")
    }
}

tasks.named<JavaCompile>("compileTestJava") {
    options.release = 21
    options.errorprone.check("UnusedVariable", CheckSeverity.WARN)
}

checkstyle {
    toolVersion = "14.4.0"
    configFile = file("config/checkstyle.xml")
    maxWarnings = 0
}

tasks.named("checkstyleTest") {
    enabled = false
}

tasks.test {
    useJUnitPlatform()
}
