package dev.komar.agent.channel;

public final class JsonObject {

    private static final char LAST_CONTROL_CHAR = 0x1f;

    private final StringBuilder out = new StringBuilder("{");

    public JsonObject put(String key, String value) {
        key(key);
        quote(value);
        return this;
    }

    public JsonObject put(String key, long value) {
        key(key);
        out.append(value);
        return this;
    }

    @Override
    public String toString() {
        return out + "}";
    }

    private void key(String key) {
        if (out.length() > 1) {
            out.append(',');
        }
        quote(key);
        out.append(':');
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
