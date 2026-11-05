package dev.komar.agent.probe;

import dev.komar.agent.spec.Limits;
import dev.komar.agent.spec.ProbeSpec;
import dev.komar.agent.spec.WhatItem;
import dev.komar.agent.value.Capture;
import java.util.LinkedHashMap;
import java.util.Map;

final class What {

    private static final Object ABSENT = Absent.INSTANCE;

    private enum Absent { INSTANCE }

    private What() {
    }

    static Map<String, Object> capture(ProbeSpec spec, Event e, Site site) {
        Limits l = spec.limits();
        Capture capture = new Capture(l.depth(), l.items(), l.string(), l.fields());
        Map<String, Object> out = new LinkedHashMap<>();
        for (WhatItem item : spec.what()) {
            Object v = item(item.expr(), e, site, capture);
            if (!(v instanceof Absent)) {
                out.put(item.key(), v);
            }
        }
        return out;
    }

    private static Object item(String expr, Event e, Site site, Capture capture) {
        Object v = ABSENT;
        switch (expr) {
            case "args":
                v = capture.each(e.args());
                break;
            case "this":
                v = e.self() == null ? ABSENT : capture.value(e.self());
                break;
            case "ret":
                v = e.hasRet() && site.returnsValue() ? capture.value(e.ret()) : ABSENT;
                break;
            case "exception":
                v = e.thrown() == null ? ABSENT : capture.value(e.thrown());
                break;
            case "duration":
                v = Double.isNaN(e.durationMs()) ? ABSENT : (Object) e.durationMs();
                break;
            default:
                break;
        }
        return v;
    }
}
