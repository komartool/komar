package dev.komar.agent.probe;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

final class Scope {

    private final Map<String, Object> values = new HashMap<>();
    private final Object[] args;

    Scope(Site site, Event e) {
        this.args = e.args();
        values.put("args", Arrays.asList(e.args()));
        values.put("vars", site.vars(e.args()));
        if (e.self() != null) {
            values.put("this", e.self());
        }
        if (e.hasRet() && site.returnsValue()) {
            values.put("ret", e.ret());
        }
        if (e.thrown() != null) {
            values.put("exception", e.thrown());
        }
        if (!Double.isNaN(e.durationMs())) {
            values.put("duration", e.durationMs());
        }
    }

    boolean hasAll(List<String> roots) {
        for (String root : roots) {
            if (!values.containsKey(root)) {
                return false;
            }
        }
        return true;
    }

    Object[] args() {
        return args;
    }

    Object get(String root) {
        return values.get(root);
    }

    Map<String, Object> roots() {
        return values;
    }
}
