package dev.komar.agent;

import static com.tngtech.archunit.library.Architectures.layeredArchitecture;
import static com.tngtech.archunit.lang.syntax.ArchRuleDefinition.noClasses;

import com.tngtech.archunit.core.importer.ImportOption;
import com.tngtech.archunit.junit.AnalyzeClasses;
import com.tngtech.archunit.junit.ArchTest;
import com.tngtech.archunit.lang.ArchRule;

@AnalyzeClasses(packages = "dev.komar.agent", importOptions = ImportOption.DoNotIncludeTests.class)
final class ArchitectureTest {

    @ArchTest
    static final ArchRule LAYERS = layeredArchitecture()
            .consideringOnlyDependenciesInLayers()
            .withOptionalLayers(true)
            .layer("Entry").definedBy("dev.komar.agent")
            .layer("Probe").definedBy("dev.komar.agent.probe..")
            .layer("Cel").definedBy("dev.komar.agent.cel..")
            .layer("Value").definedBy("dev.komar.agent.value..")
            .layer("Spec").definedBy("dev.komar.agent.spec..")
            .layer("Channel").definedBy("dev.komar.agent.channel..")
            .layer("Json").definedBy("dev.komar.agent.json..")
            .whereLayer("Entry").mayNotBeAccessedByAnyLayer()
            .whereLayer("Probe").mayOnlyBeAccessedByLayers("Entry")
            .whereLayer("Cel").mayOnlyBeAccessedByLayers("Probe")
            .whereLayer("Value").mayOnlyBeAccessedByLayers("Probe", "Cel")
            .whereLayer("Json").mayOnlyBeAccessedByLayers("Probe", "Spec", "Value")
            .whereLayer("Channel").mayOnlyBeAccessedByLayers("Entry", "Probe")
            .whereLayer("Spec").mayOnlyBeAccessedByLayers("Entry", "Probe", "Cel");

    @ArchTest
    static final ArchRule BYTEBUDDY_ONLY_IN_PROBE = noClasses()
            .that().resideOutsideOfPackages("dev.komar.agent", "dev.komar.agent.probe..")
            .should().dependOnClassesThat().resideInAnyPackage("net.bytebuddy..")
            .allowEmptyShould(true);

    @ArchTest
    static final ArchRule CEL_ONLY_IN_CEL = noClasses()
            .that().resideOutsideOfPackage("dev.komar.agent.cel..")
            .should().dependOnClassesThat().resideInAnyPackage("dev.cel..")
            .allowEmptyShould(true);

    @ArchTest
    static final ArchRule NO_STANDARD_STREAMS = noClasses()
            .should().accessField(System.class, "out")
            .orShould().accessField(System.class, "err")
            .allowEmptyShould(true);
}
