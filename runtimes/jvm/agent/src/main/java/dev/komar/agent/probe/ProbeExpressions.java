package dev.komar.agent.probe;

import dev.komar.agent.cel.CelEngine;
import dev.komar.agent.cel.ExpressionException;
import dev.komar.agent.spec.Limits;
import dev.komar.agent.spec.ProbeSpec;
import dev.komar.agent.spec.WhatItem;
import dev.komar.agent.value.Capture;
import dev.komar.agent.value.Errors;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicLong;

final class ProbeExpressions {

    private final ProbeSpec spec;
    private final Compiled when;
    private final List<Compiled> what = new ArrayList<>();
    private final AtomicLong whenErrors = new AtomicLong();
    private final AtomicBoolean whenErrorReported = new AtomicBoolean();
    private final AtomicBoolean compiled = new AtomicBoolean();

    ProbeExpressions(ProbeSpec spec, CelEngine cel) {
        this.spec = spec;
        this.when = spec.when() == null ? null : Compiled.of(spec.when(), cel);
        for (WhatItem item : spec.what()) {
            what.add(Compiled.of(item, cel));
        }
    }

    void precompile() {
        if (when != null) {
            when.error();
        }
        for (Compiled c : what) {
            c.error();
        }
    }

    private void reportCompileErrors(Recorder recorder) {
        if (!compiled.compareAndSet(false, true)) {
            return;
        }
        if (when != null && when.error() != null) {
            recorder.diag("error", spec.id(), "when could not be compiled: " + when.error());
        }
        for (Compiled c : what) {
            if (c.error() != null) {
                recorder.diag("error", spec.id(), "what \"" + c.key() + "\" could not be compiled: " + c.error());
            }
        }
    }

    Set<String> referencedVars() {
        Set<String> names = new LinkedHashSet<>();
        if (when != null) {
            names.addAll(when.vars());
        }
        for (Compiled c : what) {
            names.addAll(c.vars());
        }
        return names;
    }

    long whenErrors() {
        return whenErrors.get();
    }

    boolean test(Site site, Scope scope, Recorder recorder) {
        reportCompileErrors(recorder);
        if (when == null) {
            return true;
        }
        if (when.error() != null || !when.applies(scope, site)) {
            return false;
        }
        try {
            Object result = when.evaluate(scope);
            if (result instanceof Boolean) {
                return (Boolean) result;
            }
            failed(site, "when returned " + (result == null ? "null" : result.getClass().getSimpleName())
                    + ", not a boolean", recorder);
        } catch (ExpressionException e) {
            failed(site, e.getMessage(), recorder);
        }
        return false;
    }

    private void failed(Site site, String msg, Recorder recorder) {
        whenErrors.incrementAndGet();
        if (whenErrorReported.compareAndSet(false, true)) {
            Map<String, Object> at = new LinkedHashMap<>();
            at.put("method", site.label());
            at.put("thread", Thread.currentThread().getName());
            recorder.diag("warn", spec.id(), "when failed: " + msg + "; counted as false", "first_at", at);
        }
    }

    Map<String, Object> capture(Site site, Scope scope) {
        Limits l = spec.limits();
        Capture capture = new Capture(l.depth(), l.items(), l.string(), l.fields());
        Map<String, Object> out = new LinkedHashMap<>();
        for (Compiled c : what) {
            if (c.error() != null) {
                out.put(c.key(), Errors.message(c.error()));
            } else if (c.applies(scope, site)) {
                out.put(c.key(), value(c, scope, capture));
            }
        }
        return out;
    }

    private static Object value(Compiled c, Scope scope, Capture capture) {
        if (c.isRoot("args")) {
            return capture.each(scope.args());
        }
        try {
            return capture.value(c.evaluate(scope));
        } catch (ExpressionException e) {
            return Errors.message(e.getMessage());
        }
    }
}
