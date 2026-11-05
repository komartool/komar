package dev.komar.agent.probe;

import java.util.Arrays;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class Scope {

    private final Map<String, Object> values = new HashMap<>();
    private final Map<String, Object> vars;
    private final Map<String, Object> call;
    private final Object[] args;

    Scope(Site site, Event e) {
        this.args = e.args();
        this.vars = site.vars(e.args());
        if (e.locals() != null) {
            vars.putAll(e.locals());
        }
        values.put("args", Arrays.asList(e.args()));
        values.put("vars", vars);
        this.call = e.isCallSite() ? callValues(e) : null;
        if (call != null) {
            values.put("call", call);
        }
        if (e.self() != null) {
            values.put("this", e.self());
        }
        if (e.hasRet()) {
            values.put("ret", e.ret());
        }
        if (e.thrown() != null) {
            values.put("exception", e.thrown());
        }
        if (!Double.isNaN(e.durationMs())) {
            values.put("duration", e.durationMs());
        }
    }

    private static Map<String, Object> callValues(Event e) {
        Map<String, Object> out = new LinkedHashMap<>();
        if (e.target() != null) {
            out.put("target", e.target());
        }
        out.put("args", Arrays.asList(e.callArgs()));
        return out;
    }

    boolean hasAll(List<String> roots) {
        for (String root : roots) {
            if (!values.containsKey(root)) {
                return false;
            }
        }
        return true;
    }

    boolean hasVars(List<String> names) {
        return vars.keySet().containsAll(names);
    }

    boolean hasCallFields(List<String> fields) {
        return fields.isEmpty() || (call != null && call.keySet().containsAll(fields));
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
