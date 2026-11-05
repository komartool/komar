package dev.komar.agent.value;

import java.util.Collections;
import java.util.Map;

public final class Errors {

    private Errors() {
    }

    public static Map<String, Object> of(Throwable e) {
        String msg = e.getClass().getName();
        if (e.getMessage() != null) {
            msg += ": " + e.getMessage();
        }
        return message(msg);
    }

    public static Map<String, Object> message(String msg) {
        return Collections.<String, Object>singletonMap("@error", msg);
    }
}
