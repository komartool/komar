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
        return Collections.<String, Object>singletonMap("@error", msg);
    }
}
