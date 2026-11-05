package dev.komar.agent.spec;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;

final class Node {

    private final Map<?, ?> map;

    Node(Object value) {
        if (!(value instanceof Map)) {
            throw new IllegalArgumentException("session: expected an object");
        }
        this.map = (Map<?, ?>) value;
    }

    String string(String key) {
        Object v = map.get(key);
        return v == null ? null : String.valueOf(v);
    }

    int integer(String key) {
        Object v = map.get(key);
        if (!(v instanceof Number)) {
            throw new IllegalArgumentException("session: " + key + " must be a number");
        }
        return ((Number) v).intValue();
    }

    boolean has(String key) {
        return map.get(key) != null;
    }

    boolean bool(String key) {
        return Boolean.TRUE.equals(map.get(key));
    }

    Node node(String key) {
        return new Node(map.get(key));
    }

    List<Node> nodes(String key) {
        List<Node> out = new ArrayList<>();
        for (Object item : list(key)) {
            out.add(new Node(item));
        }
        return out;
    }

    List<String> strings(String key) {
        Object v = map.get(key);
        if (v == null) {
            return null;
        }
        List<String> out = new ArrayList<>();
        for (Object item : list(key)) {
            out.add(String.valueOf(item));
        }
        return out;
    }

    private List<?> list(String key) {
        Object v = map.get(key);
        if (v == null) {
            return Collections.emptyList();
        }
        if (!(v instanceof List)) {
            throw new IllegalArgumentException("session: " + key + " must be a list");
        }
        return (List<?>) v;
    }
}
