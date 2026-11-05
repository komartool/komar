package dev.komar.agent.probe;

import dev.komar.agent.spec.ProbeSpec;
import dev.komar.agent.spec.SessionSpec;
import java.lang.instrument.Instrumentation;
import java.security.ProtectionDomain;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import net.bytebuddy.agent.builder.AgentBuilder;
import net.bytebuddy.asm.Advice;
import net.bytebuddy.description.method.MethodDescription;
import net.bytebuddy.description.method.ParameterDescription;
import net.bytebuddy.description.type.TypeDescription;
import net.bytebuddy.dynamic.DynamicType;
import net.bytebuddy.matcher.ElementMatchers;
import net.bytebuddy.utility.JavaModule;

public final class Placement {

    private final Recorder recorder;
    private final List<ProbeRuntime> probes = new ArrayList<>();
    private final Engine engine;
    private final Set<String> failedTypes = ConcurrentHashMap.newKeySet();

    private Placement(SessionSpec spec, Recorder recorder) {
        this.recorder = recorder;
        for (ProbeSpec p : spec.probes()) {
            probes.add(new ProbeRuntime(p));
        }
        this.engine = new Engine(recorder, probes);
    }

    public static Placement install(Instrumentation inst, SessionSpec spec, Recorder recorder) {
        Placement placement = new Placement(spec, recorder);
        Hooks.install(placement.engine);
        safeBuilder()
                .disableClassFormatChanges()
                .with(AgentBuilder.RedefinitionStrategy.RETRANSFORMATION)
                .with(AgentBuilder.InjectionStrategy.Disabled.INSTANCE)
                .with(placement.listener())
                .ignore(ElementMatchers.any(), ElementMatchers.isBootstrapClassLoader())
                .or(ElementMatchers.nameStartsWith("dev.komar."))
                .or(ElementMatchers.isSynthetic())
                .type(placement::matchesAnyProbe)
                .transform(placement::transform)
                .installOn(inst);
        return placement;
    }

    private static AgentBuilder safeBuilder() {
        String property = AgentBuilder.class.getPackage().getName().replace(".agent.builder", ".safe");
        String previous = System.getProperty(property);
        System.setProperty(property, "true");
        try {
            return new AgentBuilder.Default();
        } finally {
            if (previous == null) {
                System.clearProperty(property);
            } else {
                System.setProperty(property, previous);
            }
        }
    }

    public void finish() {
        engine.finish();
    }

    private boolean matchesAnyProbe(TypeDescription td) {
        boolean any = false;
        for (ProbeRuntime p : probes) {
            any |= p.matchesType(td);
        }
        return any;
    }

    private DynamicType.Builder<?> transform(DynamicType.Builder<?> builder, TypeDescription td,
                                             ClassLoader loader, JavaModule module, ProtectionDomain domain) {
        DynamicType.Builder<?> result = builder;
        List<ProbeRuntime> forType = new ArrayList<>();
        for (ProbeRuntime p : probes) {
            if (p.matchesType(td)) {
                forType.add(p);
            }
        }
        int[] counts = new int[forType.size()];
        for (MethodDescription.InDefinedShape m : td.getDeclaredMethods()) {
            List<ProbeRuntime> matched = new ArrayList<>();
            for (int i = 0; i < forType.size(); i++) {
                ProbeRuntime p = forType.get(i);
                if (p.matchesMethod(m) && p.admitClass(td.getName(), recorder)) {
                    matched.add(p);
                    counts[i]++;
                }
            }
            if (!matched.isEmpty()) {
                result = advise(result, td, m, matched);
            }
        }
        reportPlaced(td, forType, counts);
        return result;
    }

    private DynamicType.Builder<?> advise(DynamicType.Builder<?> builder, TypeDescription td,
                                          MethodDescription.InDefinedShape m, List<ProbeRuntime> matched) {
        boolean returnsValue = !m.isConstructor() && !m.getReturnType().represents(void.class);
        int site = engine.register(new Site(label(td, m), returnsValue, matched));
        Class<?> advice = m.isConstructor() ? ConstructorAdvice.class : MethodAdvice.class;
        return builder.visit(Advice.withCustomMapping()
                .bind(SiteId.class, site)
                .to(advice)
                .on(ElementMatchers.is(m)));
    }

    private void reportPlaced(TypeDescription td, List<ProbeRuntime> forType, int[] counts) {
        for (int i = 0; i < forType.size(); i++) {
            if (counts[i] > 0) {
                forType.get(i).placed(counts[i]);
                recorder.diag("info", forType.get(i).id(), "placed",
                        "class", td.getName(), "methods", (long) counts[i]);
            }
        }
    }

    private static String label(TypeDescription td, MethodDescription m) {
        StringBuilder sb = new StringBuilder(td.getName()).append('.');
        sb.append(m.isConstructor() ? ProbeSpec.CONSTRUCTOR : m.getName()).append('(');
        boolean first = true;
        for (ParameterDescription p : m.getParameters()) {
            if (!first) {
                sb.append(", ");
            }
            sb.append(p.getType().asErasure().getSimpleName());
            first = false;
        }
        return sb.append(')').toString();
    }

    private AgentBuilder.Listener listener() {
        return new AgentBuilder.Listener.Adapter() {
            @Override
            public void onError(String typeName, ClassLoader classLoader, JavaModule module,
                                boolean loaded, Throwable throwable) {
                if (failedTypes.add(typeName)) {
                    recorder.diag("error", null, "could not instrument " + typeName + ": " + throwable,
                            "class", typeName);
                }
            }
        };
    }
}
