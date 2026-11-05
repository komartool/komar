package dev.komar.agent.json;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public final class JsonReader {

    private static final int HEX_RADIX = 16;
    private static final int UNICODE_ESCAPE_LENGTH = 4;
    private static final String ESCAPES = "ntrbf";
    private static final String UNESCAPED = "\n\t\r\b\f";

    private final String text;
    private int pos;

    private JsonReader(String text) {
        this.text = text;
    }

    public static Object parse(String text) {
        JsonReader r = new JsonReader(text);
        Object value = r.value();
        r.skipSpace();
        if (r.pos != text.length()) {
            throw r.error("trailing characters");
        }
        return value;
    }

    private Object value() {
        skipSpace();
        if (pos >= text.length()) {
            throw error("unexpected end");
        }
        char c = text.charAt(pos);
        if (c == '{') {
            return object();
        }
        if (c == '[') {
            return array();
        }
        if (c == '"') {
            return string();
        }
        return literal();
    }

    private Map<String, Object> object() {
        Map<String, Object> map = new LinkedHashMap<>();
        pos++;
        skipSpace();
        if (peek('}')) {
            return map;
        }
        do {
            skipSpace();
            String key = string();
            skipSpace();
            expect(':');
            map.put(key, value());
            skipSpace();
        } while (consume(','));
        expect('}');
        return map;
    }

    private List<Object> array() {
        List<Object> list = new ArrayList<>();
        pos++;
        skipSpace();
        if (peek(']')) {
            return list;
        }
        do {
            list.add(value());
            skipSpace();
        } while (consume(','));
        expect(']');
        return list;
    }

    private String string() {
        expect('"');
        StringBuilder sb = new StringBuilder();
        while (pos < text.length()) {
            char c = text.charAt(pos++);
            if (c == '"') {
                return sb.toString();
            }
            sb.append(c == '\\' ? unescape() : c);
        }
        throw error("unterminated string");
    }

    private char unescape() {
        char c = text.charAt(pos++);
        if (c == 'u') {
            pos += UNICODE_ESCAPE_LENGTH;
            return (char) Integer.parseInt(text.substring(pos - UNICODE_ESCAPE_LENGTH, pos), HEX_RADIX);
        }
        int i = ESCAPES.indexOf(c);
        return i < 0 ? c : UNESCAPED.charAt(i);
    }

    private Object literal() {
        int start = pos;
        while (pos < text.length() && "{}[],: \t\r\n".indexOf(text.charAt(pos)) < 0) {
            pos++;
        }
        String word = text.substring(start, pos);
        switch (word) {
            case "true":
                return true;
            case "false":
                return false;
            case "null":
                return null;
            default:
                return number(word);
        }
    }

    private Object number(String word) {
        try {
            if (word.contains(".") || word.contains("e") || word.contains("E")) {
                return Double.valueOf(word);
            }
            return Long.valueOf(word);
        } catch (NumberFormatException e) {
            throw error("invalid literal " + word);
        }
    }

    private void skipSpace() {
        while (pos < text.length() && Character.isWhitespace(text.charAt(pos))) {
            pos++;
        }
    }

    private boolean peek(char c) {
        if (pos < text.length() && text.charAt(pos) == c) {
            pos++;
            return true;
        }
        return false;
    }

    private boolean consume(char c) {
        return peek(c);
    }

    private void expect(char c) {
        if (!peek(c)) {
            throw error("expected '" + c + "'");
        }
    }

    private IllegalArgumentException error(String msg) {
        return new IllegalArgumentException("invalid JSON at " + pos + ": " + msg);
    }
}
