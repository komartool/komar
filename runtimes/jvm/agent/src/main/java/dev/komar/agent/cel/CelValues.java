package dev.komar.agent.cel;

import dev.cel.common.values.NullValue;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Collection;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.RandomAccess;

final class CelValues {

    private static final int LONG_BITS = 63;

    private CelValues() {
    }

    static Object wrap(Object v) {
        if (v == null) {
            return NullValue.NULL_VALUE;
        }
        if (v instanceof Long || v instanceof Double || v instanceof Boolean || v instanceof String) {
            return v;
        }
        Object number = number(v);
        return number != null ? number : structure(v);
    }

    private static Object number(Object v) {
        if (v instanceof Integer || v instanceof Short || v instanceof Byte) {
            return ((Number) v).longValue();
        }
        if (v instanceof Float || v instanceof BigDecimal) {
            return ((Number) v).doubleValue();
        }
        if (v instanceof BigInteger && isJdk(v)) {
            BigInteger b = (BigInteger) v;
            return b.bitLength() <= LONG_BITS ? (Object) b.longValue() : (Object) b.doubleValue();
        }
        return null;
    }

    private static Object structure(Object v) {
        if (v.getClass().isArray()) {
            return new LazyList(v);
        }
        if (!isJdk(v)) {
            return new CelObject(v);
        }
        return jdk(v);
    }

    private static Object jdk(Object v) {
        if (v instanceof List && v instanceof RandomAccess) {
            return new LazyList(v);
        }
        if (v instanceof Collection) {
            return new LazyList(new ArrayList<>((Collection<?>) v));
        }
        if (v instanceof Map) {
            Map<Object, Object> out = new LinkedHashMap<>();
            for (Map.Entry<?, ?> e : ((Map<?, ?>) v).entrySet()) {
                out.put(wrap(e.getKey()), wrap(e.getValue()));
            }
            return out;
        }
        return text(v);
    }

    static Object root(String name, Object value) {
        if (value instanceof Map && ("vars".equals(name) || "call".equals(name))) {
            return new LazyMap((Map<?, ?>) value);
        }
        return wrap(value);
    }

    private static Object text(Object v) {
        if (v instanceof CharSequence || v instanceof Character || v instanceof Enum) {
            return v instanceof Enum ? ((Enum<?>) v).name() : v.toString();
        }
        if (v.getClass().getName().startsWith("java.time.") || v instanceof java.util.UUID) {
            return v.toString();
        }
        return new CelObject(v);
    }

    static Object unwrap(Object v) {
        if (v instanceof CelObject) {
            return ((CelObject) v).target();
        }
        if (v instanceof NullValue) {
            return null;
        }
        return lazySource(v);
    }

    private static Object lazySource(Object v) {
        if (v instanceof LazyList) {
            return ((LazyList) v).source();
        }
        if (v instanceof LazyMap) {
            return ((LazyMap) v).source();
        }
        return v instanceof List || v instanceof Map ? container(v) : v;
    }

    private static Object container(Object v) {
        if (v instanceof List) {
            List<Object> out = new ArrayList<>();
            for (Object item : (List<?>) v) {
                out.add(unwrap(item));
            }
            return out;
        }
        Map<Object, Object> out = new LinkedHashMap<>();
        for (Map.Entry<?, ?> e : ((Map<?, ?>) v).entrySet()) {
            out.put(unwrap(e.getKey()), unwrap(e.getValue()));
        }
        return out;
    }

    private static boolean isJdk(Object v) {
        return v.getClass().getClassLoader() == null;
    }
}
