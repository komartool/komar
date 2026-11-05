package dev.komar.agent.json;

import java.util.List;
import java.util.Map;

public final class JsonWriter {

    private static final char LAST_CONTROL_CHAR = 0x1f;

    private final StringBuilder out = new StringBuilder();

    public static String toJson(Object value) {
        return new JsonWriter().value(value).toString();
    }

    public JsonWriter beginObject() {
        separate();
        out.append('{');
        return this;
    }

    public JsonWriter endObject() {
        out.append('}');
        return this;
    }

    public JsonWriter beginArray() {
        separate();
        out.append('[');
        return this;
    }

    public JsonWriter endArray() {
        out.append(']');
        return this;
    }

    public JsonWriter name(String name) {
        separate();
        quote(name);
        out.append(':');
        return this;
    }

    public JsonWriter field(String name, Object value) {
        return name(name).value(value);
    }

    public JsonWriter value(Object v) {
        if (v instanceof JsonValue) {
            ((JsonValue) v).writeJson(this);
        } else if (v instanceof Map) {
            map((Map<?, ?>) v);
        } else if (v instanceof List) {
            list((List<?>) v);
        } else {
            scalar(v);
        }
        return this;
    }

    @Override
    public String toString() {
        return out.toString();
    }

    private void map(Map<?, ?> map) {
        beginObject();
        for (Map.Entry<?, ?> e : map.entrySet()) {
            name(String.valueOf(e.getKey()));
            value(e.getValue());
        }
        endObject();
    }

    private void list(List<?> list) {
        beginArray();
        for (Object item : list) {
            value(item);
        }
        endArray();
    }

    private void scalar(Object v) {
        separate();
        if (v == null) {
            out.append("null");
        } else if (v instanceof Boolean || v instanceof Long || v instanceof Integer) {
            out.append(v);
        } else if (v instanceof Double && isFinite((Double) v)) {
            out.append(v);
        } else {
            quote(String.valueOf(v));
        }
    }

    private static boolean isFinite(double d) {
        return !Double.isNaN(d) && !Double.isInfinite(d);
    }

    private void separate() {
        if (out.length() == 0) {
            return;
        }
        char last = out.charAt(out.length() - 1);
        if (last != '{' && last != '[' && last != ':') {
            out.append(',');
        }
    }

    private void quote(String s) {
        out.append('"');
        for (int i = 0; i < s.length(); i++) {
            escape(s.charAt(i));
        }
        out.append('"');
    }

    private void escape(char c) {
        switch (c) {
            case '"':
                out.append("\\\"");
                break;
            case '\\':
                out.append("\\\\");
                break;
            case '\n':
                out.append("\\n");
                break;
            case '\r':
                out.append("\\r");
                break;
            case '\t':
                out.append("\\t");
                break;
            default:
                if (c <= LAST_CONTROL_CHAR) {
                    out.append(String.format("\\u%04x", (int) c));
                } else {
                    out.append(c);
                }
        }
    }
}
