package dev.komar.agent.probe;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class CallPoint {

    private final Site site;
    private final String[] names;
    private final List<ProbeRuntime> probes;
    private final boolean returnsValue;
    private final boolean isStatic;

    CallPoint(Site site, String[] names, List<ProbeRuntime> probes, boolean returnsValue, boolean isStatic) {
        this.isStatic = isStatic;
        this.site = site;
        this.names = names.clone();
        this.probes = Collections.unmodifiableList(new ArrayList<>(probes));
        this.returnsValue = returnsValue;
    }

    Site site() {
        return site;
    }

    boolean returnsValue() {
        return returnsValue;
    }

    List<ProbeRuntime> matching(Object target) {
        if (isStatic) {
            return probes;
        }
        if (target == null) {
            return Collections.emptyList();
        }
        List<ProbeRuntime> out = new ArrayList<>(probes.size());
        for (ProbeRuntime p : probes) {
            if (p.matchesReceiver(target.getClass())) {
                out.add(p);
            }
        }
        return Collections.unmodifiableList(out);
    }

    Map<String, Object> locals(Object[] values) {
        Map<String, Object> out = new LinkedHashMap<>();
        for (int i = 0; i < names.length && i < values.length; i++) {
            out.put(names[i], values[i]);
        }
        return out;
    }
}
