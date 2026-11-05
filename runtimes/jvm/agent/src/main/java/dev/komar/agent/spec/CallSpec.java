package dev.komar.agent.spec;

import java.util.Collections;
import java.util.List;

public final class CallSpec {

    private final String classPattern;
    private final boolean subtypes;
    private final String method;
    private final List<String> params;

    private CallSpec(Node n) {
        this.classPattern = n.string("class");
        this.subtypes = n.bool("subtypes");
        this.method = n.string("method");
        List<String> p = n.strings("params");
        this.params = p == null ? null : Collections.unmodifiableList(p);
    }

    static CallSpec from(Node n) {
        return new CallSpec(n);
    }

    public String classPattern() {
        return classPattern;
    }

    public boolean subtypes() {
        return subtypes;
    }

    public String method() {
        return method;
    }

    public List<String> params() {
        return params;
    }
}
