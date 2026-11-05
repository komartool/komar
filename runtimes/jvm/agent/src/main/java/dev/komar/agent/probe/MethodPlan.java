package dev.komar.agent.probe;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Set;
import java.util.TreeSet;
import org.objectweb.asm.tree.MethodInsnNode;

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

    List<ProbeRuntime> lineProbes() {
        List<ProbeRuntime> out = new ArrayList<>();
        for (ProbeRuntime p : probes) {
            if (p.spec().isLine()) {
                out.add(p);
            }
        }
        return out;
    }

    boolean hasCalls() {
        for (ProbeRuntime p : probes) {
            if (p.spec().isCall()) {
                return true;
            }
        }
        return false;
    }

    Set<Integer> lines() {
        Set<Integer> out = new TreeSet<>();
        for (ProbeRuntime p : lineProbes()) {
            out.addAll(p.spec().lines());
        }
        return out;
    }

    List<ProbeRuntime> probesAt(int line) {
        List<ProbeRuntime> out = new ArrayList<>();
        for (ProbeRuntime p : lineProbes()) {
            if (p.spec().lines().contains(line)) {
                out.add(p);
            }
        }
        return out;
    }

    List<ProbeRuntime> probesCalling(MethodInsnNode insn) {
        List<ProbeRuntime> out = new ArrayList<>();
        for (ProbeRuntime p : probes) {
            if (p.spec().isCall() && p.matchesCallInsn(insn)) {
                out.add(p);
            }
        }
        return out;
    }
}
