package dev.komar.agent.probe;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class Site {

    private final String label;
    private final String enclosing;
    private final boolean returnsValue;
    private final String[] paramNames;
    private final List<ProbeRuntime> probes;

    Site(String label, String enclosing, boolean returnsValue, String[] paramNames, List<ProbeRuntime> probes) {
        this.label = label;
        this.enclosing = enclosing;
        this.returnsValue = returnsValue;
        this.paramNames = paramNames.clone();
        this.probes = probes;
    }

    String label() {
        return label;
    }

    String enclosing() {
        return enclosing;
    }

    boolean returnsValue() {
        return returnsValue;
    }

    List<ProbeRuntime> probes() {
        return probes;
    }

    boolean wants(String point) {
        for (ProbeRuntime p : probes) {
            if (p.wants(point)) {
                return true;
            }
        }
        return false;
    }

    boolean hasVar(String name) {
        for (String p : paramNames) {
            if (name.equals(p)) {
                return true;
            }
        }
        return false;
    }

    boolean namesKnown() {
        for (String p : paramNames) {
            if (p == null) {
                return false;
            }
        }
        return true;
    }

    Map<String, Object> vars(Object[] args) {
        Map<String, Object> out = new LinkedHashMap<>();
        for (int i = 0; i < paramNames.length && i < args.length; i++) {
            if (paramNames[i] != null) {
                out.put(paramNames[i], args[i]);
            }
        }
        return out;
    }
}
