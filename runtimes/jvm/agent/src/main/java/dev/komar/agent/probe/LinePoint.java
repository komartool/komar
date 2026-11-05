package dev.komar.agent.probe;

import java.util.LinkedHashMap;
import java.util.Map;

final class LinePoint {

    private final Site site;
    private final int line;
    private final String[] names;

    LinePoint(Site site, int line, String[] names) {
        this.site = site;
        this.line = line;
        this.names = names.clone();
    }

    Site site() {
        return site;
    }

    int line() {
        return line;
    }

    Map<String, Object> locals(Object[] values) {
        Map<String, Object> out = new LinkedHashMap<>();
        for (int i = 0; i < names.length && i < values.length; i++) {
            out.put(names[i], values[i]);
        }
        return out;
    }
}
