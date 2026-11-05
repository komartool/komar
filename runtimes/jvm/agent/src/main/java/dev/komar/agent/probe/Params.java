package dev.komar.agent.probe;

import java.util.List;
import net.bytebuddy.description.method.ParameterDescription;
import net.bytebuddy.description.method.ParameterList;
import net.bytebuddy.description.type.TypeDefinition;
import net.bytebuddy.description.type.TypeDescription;

final class Params {

    private static final String ARRAY = "[]";
    private static final String VARARGS = "...";

    private Params() {
    }

    static boolean match(List<String> written, ParameterList<?> actual) {
        if (written.size() != actual.size()) {
            return false;
        }
        for (int i = 0; i < written.size(); i++) {
            if (!match(written.get(i), actual.get(i))) {
                return false;
            }
        }
        return true;
    }

    private static boolean match(String written, ParameterDescription param) {
        String w = eraseGenerics(written).replace(" ", "");
        if (w.endsWith(VARARGS)) {
            w = w.substring(0, w.length() - VARARGS.length()) + ARRAY;
        }
        TypeDescription.Generic type = param.getType();
        while (w.endsWith(ARRAY)) {
            if (!type.isArray()) {
                return false;
            }
            type = type.getComponentType();
            w = w.substring(0, w.length() - ARRAY.length());
        }
        return !type.isArray() && matchBase(w, type);
    }

    private static boolean matchBase(String w, TypeDescription.Generic type) {
        if (type.getSort() == TypeDefinition.Sort.VARIABLE || type.getSort() == TypeDefinition.Sort.VARIABLE_SYMBOLIC) {
            return type.getSymbol().equals(w);
        }
        TypeDescription erased = type.asErasure();
        if (w.contains(".")) {
            return erased.getName().equals(w) || erased.getName().replace('$', '.').equals(w);
        }
        return erased.getSimpleName().equals(w);
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
