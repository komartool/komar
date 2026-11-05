package dev.komar.agent.probe;

import java.util.List;
import org.objectweb.asm.Type;

final class CallParams {

    private static final String ARRAY = "[]";
    private static final String VARARGS = "...";

    private CallParams() {
    }

    static boolean match(List<String> written, Type[] actual) {
        if (written.size() != actual.length) {
            return false;
        }
        for (int i = 0; i < actual.length; i++) {
            if (!match(written.get(i), actual[i].getClassName())) {
                return false;
            }
        }
        return true;
    }

    private static boolean match(String written, String actual) {
        String w = eraseGenerics(written).replace(" ", "");
        if (w.endsWith(VARARGS)) {
            w = w.substring(0, w.length() - VARARGS.length()) + ARRAY;
        }
        String a = actual;
        while (w.endsWith(ARRAY) && a.endsWith(ARRAY)) {
            w = w.substring(0, w.length() - ARRAY.length());
            a = a.substring(0, a.length() - ARRAY.length());
        }
        if (w.endsWith(ARRAY) || a.endsWith(ARRAY)) {
            return false;
        }
        if (w.contains(".")) {
            return a.equals(w) || a.replace('$', '.').equals(w);
        }
        return simpleName(a).equals(w);
    }

    private static String simpleName(String name) {
        int cut = Math.max(name.lastIndexOf('.'), name.lastIndexOf('$'));
        return name.substring(cut + 1);
    }

    private static String eraseGenerics(String s) {
        StringBuilder sb = new StringBuilder();
        int depth = 0;
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            if (c == '<') {
                depth++;
            } else if (c == '>') {
                depth--;
            } else if (depth == 0) {
                sb.append(c);
            }
        }
        return sb.toString();
    }
}
