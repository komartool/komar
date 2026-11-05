package dev.komar.agent.probe;

import dev.komar.agent.cel.CelEngine;
import dev.komar.agent.spec.ProbeSpec;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import java.util.TreeSet;
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
    private final Set<Integer> placedLines = ConcurrentHashMap.newKeySet();
    private volatile boolean typeSeen;
    private volatile boolean abstractSeen;
    private volatile boolean noLineTable;

    ProbeRuntime(ProbeSpec spec, CelEngine cel) {
        this.spec = spec;
        this.expressions = new ProbeExpressions(spec, cel);
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

    boolean fires(Event e) {
        if (Points.LINE.equals(e.point())) {
            return spec.lines().contains(e.line());
        }
        return wants(e.point());
    }

    boolean matchesType(TypeDescription td) {
        boolean matched = typeMatcher.matches(td) || (spec.isLine() && insideTarget(td));
        if (matched) {
            typeSeen = true;
            abstractSeen |= td.isInterface() || td.isAbstract();
        }
        return matched;
    }

    boolean matchesMethod(TypeDescription td, MethodDescription m) {
        if (m.isAbstract() || m.isNative() || m.isBridge() || m.isTypeInitializer()) {
            return false;
        }
        if (spec.isLine() && (insideTarget(td) || isTargetLambda(m))) {
            return true;
        }
        return !m.isSynthetic() && matchesDeclared(m);
    }

    private boolean isTargetLambda(MethodDescription m) {
        return m.isSynthetic() && Lambdas.isLambda(m.getName()) && targetsName(Lambdas.owner(m.getName()));
    }

    private boolean matchesDeclared(MethodDescription m) {
        if (spec.isConstructor() != m.isConstructor()) {
            return false;
        }
        if (!spec.isConstructor() && !methodPattern.matches(m.getName())) {
            return false;
        }
        return spec.params() == null || Params.match(spec.params(), m.getParameters());
    }

    private boolean insideTarget(TypeDescription td) {
        MethodDescription.InDefinedShape em = Lambdas.enclosingMethod(td);
        if (em == null) {
            return false;
        }
        TypeDescription owner = em.getDeclaringType();
        String name = em.isConstructor() ? ProbeSpec.CONSTRUCTOR : Lambdas.owner(em.getName());
        return (classPattern.matchesClass(owner.getName()) && targetsName(name)) || insideTarget(owner);
    }

    private boolean targetsName(String name) {
        return spec.isConstructor() ? ProbeSpec.CONSTRUCTOR.equals(name) : methodPattern.matches(name);
    }

    void placedLine(int line) {
        placedLines.add(line);
    }

    void noLineTable() {
        noLineTable = true;
    }

    List<String> lineProblems() {
        List<String> out = new ArrayList<>();
        if (!spec.isLine() || methods.get() == 0) {
            return out;
        }
        Set<Integer> missing = new TreeSet<>(spec.lines());
        missing.removeAll(placedLines);
        if (missing.isEmpty()) {
            return out;
        }
        if (placedLines.isEmpty() && noLineTable) {
            out.add("cannot place line probe: " + classPattern + " has no line number table (compiled with -g:none)");
            return out;
        }
        for (int line : missing) {
            out.add("line " + line + " has no code in " + methodPattern
                    + " or its lambdas and local classes (blank line, comment, or outside the method)");
        }
        return out;
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
