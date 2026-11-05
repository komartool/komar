import net.ltgt.gradle.errorprone.CheckSeverity
import net.ltgt.gradle.errorprone.errorprone

plugins {
    java
    checkstyle
    id("net.ltgt.errorprone") version "5.2.0"
    id("com.gradleup.shadow") version "9.6.1"
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
    implementation("net.bytebuddy:byte-buddy:1.18.17")
    implementation("dev.cel:compiler:0.15.0")
    implementation("dev.cel:runtime:0.15.0")
    compileOnly("com.github.spotbugs:spotbugs-annotations:4.10.5")
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

tasks.shadowJar {
    archiveFileName = "komar-agent.jar"
    relocate("net.bytebuddy", "dev.komar.shaded.bytebuddy")
    relocate("dev.cel", "dev.komar.shaded.cel")
    relocate("cel.parser", "dev.komar.shaded.celparser")
    relocate("com.google", "dev.komar.shaded.google")
    relocate("org.antlr", "dev.komar.shaded.antlr")
    relocate("org.threeten", "dev.komar.shaded.threeten")
    relocate("org.jspecify", "dev.komar.shaded.jspecify")
    relocate("org.yaml", "dev.komar.shaded.yaml")
    exclude("META-INF/maven/**", "META-INF/LICENSE*", "META-INF/NOTICE*", "module-info.class", "META-INF/versions/*/module-info.class", "google/protobuf/*.proto")
    manifest {
        attributes(
            "Premain-Class" to "dev.komar.agent.Agent",
            "Can-Retransform-Classes" to "true",
            "Multi-Release" to "true",
        )
    }
}

tasks.named("checkstyleTest") {
    enabled = false
}

tasks.test {
    useJUnitPlatform()
}
