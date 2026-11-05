package dev.komar.agent.value;

import java.lang.reflect.Array;
import java.lang.reflect.Field;
import java.util.ArrayList;
import java.util.Collection;
import java.util.IdentityHashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public final class Capture {

    private final int depth;
    private final int items;
    private final int string;
    private final int fields;
    private final IdentityHashMap<Object, ObjectNode> seen = new IdentityHashMap<>();
    private int lastId;

    public Capture(int depth, int items, int string, int fields) {
        this.depth = depth;
        this.items = items;
        this.string = string;
        this.fields = fields;
    }

    public Object value(Object v) {
        return render(v, 0);
    }

    public List<Object> each(Object[] values) {
        List<Object> out = new ArrayList<>(values.length);
        for (Object v : values) {
            out.add(render(v, 0));
        }
        return out;
    }

    int nextId() {
        return ++lastId;
    }

    private Object render(Object v, int level) {
        if (v == null) {
            return null;
        }
        Object scalar = Scalars.render(v, string);
        if (Scalars.isScalar(scalar)) {
            return scalar;
        }
        try {
            return container(v, level);
        } catch (RuntimeException | LinkageError e) {
            return Errors.of(e);
        }
    }

    private Object container(Object v, int level) {
        ObjectNode prior = seen.get(v);
        if (prior != null) {
            return prior.reference(this);
        }
        if (level >= depth) {
            return new ObjectNode(v.getClass().getTypeName()).cut();
        }
        if (v.getClass().isArray()) {
            return array(v, level);
        }
        return Scalars.isJdk(v) ? jdkObject(v, level) : object(v, level);
    }

    private Object jdkObject(Object v, int level) {
        if (v instanceof Collection) {
            return collection((Collection<?>) v, level);
        }
        if (v instanceof Map) {
            return map((Map<?, ?>) v, level);
        }
        if (v instanceof Throwable) {
            return object(v, level);
        }
        return new ObjectNode(v.getClass().getTypeName());
    }

    private Object array(Object v, int level) {
        int size = Array.getLength(v);
        List<Object> out = new ArrayList<>(Math.min(size, items));
        for (int i = 0; i < size && i < items; i++) {
            out.add(render(Array.get(v, i), level + 1));
        }
        return sized(size, out);
    }

    private Object collection(Collection<?> c, int level) {
        int size = c.size();
        List<Object> out = new ArrayList<>(Math.min(size, items));
        for (Object item : c) {
            if (out.size() >= items) {
                break;
            }
            out.add(render(item, level + 1));
        }
        return sized(size, out);
    }

    private Object map(Map<?, ?> m, int level) {
        boolean simpleKeys = true;
        for (Object k : m.keySet()) {
            simpleKeys &= k == null || Scalars.isScalar(Scalars.render(k, string));
        }
        Object out = simpleKeys ? keyedMap(m, level) : entryList(m, level);
        return sized(m.size(), out);
    }

    private Map<String, Object> keyedMap(Map<?, ?> m, int level) {
        Map<String, Object> out = new LinkedHashMap<>();
        for (Map.Entry<?, ?> e : m.entrySet()) {
            if (out.size() >= items) {
                break;
            }
            out.put(String.valueOf(e.getKey() == null ? null : Scalars.render(e.getKey(), string)),
                    render(e.getValue(), level + 1));
        }
        return out;
    }

    private List<Object> entryList(Map<?, ?> m, int level) {
        List<Object> out = new ArrayList<>();
        for (Map.Entry<?, ?> e : m.entrySet()) {
            if (out.size() >= items) {
                break;
            }
            Map<String, Object> entry = new LinkedHashMap<>();
            entry.put("@key", render(e.getKey(), level + 1));
            entry.put("@value", render(e.getValue(), level + 1));
            out.add(entry);
        }
        return out;
    }

    private Object sized(int size, Object shown) {
        if (size <= items) {
            return shown;
        }
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("@size", (long) size);
        m.put("@items", shown);
        return m;
    }

    private ObjectNode object(Object v, int level) {
        ObjectNode node = new ObjectNode(v.getClass().getTypeName());
        seen.put(v, node);
        if (v instanceof Throwable) {
            Throwables.describe((Throwable) v, node, value -> render(value, level + 1));
        }
        int shown = 0;
        for (Field f : Fields.of(v.getClass())) {
            String name = node.has(f.getName())
                    ? f.getDeclaringClass().getSimpleName() + "." + f.getName()
                    : f.getName();
            if (shown++ >= fields) {
                node.omit(name);
                continue;
            }
            node.put(name, field(f, v, level));
        }
        return node;
    }

    private Object field(Field f, Object owner, int level) {
        try {
            f.setAccessible(true);
            return render(f.get(owner), level + 1);
        } catch (IllegalAccessException | RuntimeException e) {
            return Errors.of(e);
        }
    }
}
