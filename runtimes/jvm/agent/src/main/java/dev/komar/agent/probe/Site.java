package dev.komar.agent.probe;

import java.util.List;

final class Site {

    private final String label;
    private final boolean returnsValue;
    private final List<ProbeRuntime> probes;

    Site(String label, boolean returnsValue, List<ProbeRuntime> probes) {
        this.label = label;
        this.returnsValue = returnsValue;
        this.probes = probes;
    }

    String label() {
        return label;
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
}
