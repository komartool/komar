package dev.komar.agent.probe;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Set;
import java.util.TreeSet;

final class MethodPlan {

    private final Site site;
    private final List<ProbeRuntime> probes;

    MethodPlan(Site site, List<ProbeRuntime> probes) {
        this.site = site;
        this.probes = Collections.unmodifiableList(new ArrayList<>(probes));
    }

    Site site() {
        return site;
    }

    List<ProbeRuntime> probes() {
        return probes;
    }

    Set<Integer> lines() {
        Set<Integer> out = new TreeSet<>();
        for (ProbeRuntime p : probes) {
            out.addAll(p.spec().lines());
        }
        return out;
    }

    List<ProbeRuntime> probesAt(int line) {
        List<ProbeRuntime> out = new ArrayList<>();
        for (ProbeRuntime p : probes) {
            if (p.spec().lines().contains(line)) {
                out.add(p);
            }
        }
        return out;
    }
}
