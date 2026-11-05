package dev.komar.agent.value;

import java.util.function.Function;

final class Throwables {

    private Throwables() {
    }

    static void describe(Throwable t, ObjectNode node, Function<Object, Object> render) {
        if (isJdkMethod(t, "getMessage")) {
            node.put("message", t.getMessage());
        }
        if (isJdkMethod(t, "getCause")) {
            Throwable cause = t.getCause();
            if (cause != null) {
                node.put("cause", render.apply(cause));
            }
        }
    }

    static boolean isJdkMethod(Throwable t, String name) {
        try {
            return t.getClass().getMethod(name).getDeclaringClass().getClassLoader() == null;
        } catch (NoSuchMethodException e) {
            return false;
        }
    }
}
