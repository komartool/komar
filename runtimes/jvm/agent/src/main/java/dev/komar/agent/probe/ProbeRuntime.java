package dev.komar.agent.probe;

import dev.komar.agent.cel.CelEngine;
import dev.komar.agent.spec.ProbeSpec;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
import net.bytebuddy.description.method.MethodDescription;
import net.bytebuddy.description.type.TypeDescription;
import net.bytebuddy.matcher.ElementMatcher;
import net.bytebuddy.matcher.ElementMatchers;

final class ProbeRuntime {

    private final ProbeSpec spec;
    private final ProbeExpressions expressions;
    private final NamePattern classPattern;
    private final NamePattern methodPattern;
    private final ElementMatcher<TypeDescription> typeMatcher;
    private final AtomicInteger hits = new AtomicInteger();
    private final AtomicInteger methods = new AtomicInteger();
    private final Set<String> classes = ConcurrentHashMap.newKeySet();
    private final Set<String> skipped = ConcurrentHashMap.newKeySet();
    private volatile boolean typeSeen;
    private volatile boolean abstractSeen;

    ProbeRuntime(ProbeSpec spec, CelEngine cel, Recorder recorder) {
        this.spec = spec;
        this.expressions = new ProbeExpressions(spec, cel, recorder);
        this.classPattern = new NamePattern(spec.classPattern());
        this.methodPattern = new NamePattern(spec.method());
        ElementMatcher<TypeDescription> named = td -> classPattern.matchesClass(td.getName());
        this.typeMatcher = spec.subtypes() ? ElementMatchers.hasSuperType(named) : named;
    }

    ProbeSpec spec() {
        return spec;
    }

    ProbeExpressions expressions() {
        return expressions;
    }

    String id() {
        return spec.id();
    }

    boolean wants(String point) {
        return spec.at().contains(point);
    }

    boolean matchesType(TypeDescription td) {
        boolean matched = typeMatcher.matches(td);
        if (matched) {
            typeSeen = true;
            abstractSeen |= td.isInterface() || td.isAbstract();
        }
        return matched;
    }

    boolean matchesMethod(MethodDescription m) {
        if (m.isAbstract() || m.isNative() || m.isBridge() || m.isSynthetic() || m.isTypeInitializer()) {
            return false;
        }
        if (spec.isConstructor() != m.isConstructor()) {
            return false;
        }
        if (!spec.isConstructor() && !methodPattern.matches(m.getName())) {
            return false;
        }
        return spec.params() == null || Params.match(spec.params(), m.getParameters());
    }

    boolean admitClass(String name, Recorder recorder) {
        if (classes.contains(name)) {
            return true;
        }
        if (classes.size() >= spec.limits().classes()) {
            if (skipped.add(name)) {
                recorder.diag("warn", id(), "class skipped: probe already matched " + spec.limits().classes()
                        + " classes (limits.classes)", "class", name);
            }
            return false;
        }
        classes.add(name);
        return true;
    }

    void placed(int count) {
        methods.addAndGet(count);
    }

    boolean admitHit(Recorder recorder) {
        int n = hits.incrementAndGet();
        int max = spec.limits().maxHits();
        if (n == max + 1) {
            recorder.diag("warn", id(), "max_hits reached; probe stopped");
        }
        return n <= max;
    }

    String unplacedReason() {
        if (methods.get() > 0) {
            return null;
        }
        if (!typeSeen) {
            return "no class matched " + classPattern + " (none was loaded during the run)";
        }
        String what = spec.isConstructor() ? "no constructor" : "no method " + methodPattern;
        String params = spec.params() == null ? "" : " with params " + spec.params();
        String hint = abstractSeen && !spec.subtypes()
                ? "; " + classPattern + " is an interface or abstract class,"
                        + " use subtypes: true to probe implementations"
                : "";
        return what + params + " matched in " + classPattern + hint;
    }
}
