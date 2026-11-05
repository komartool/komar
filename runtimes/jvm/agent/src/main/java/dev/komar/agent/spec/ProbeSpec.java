package dev.komar.agent.spec;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

public final class ProbeSpec {

    public static final String CONSTRUCTOR = "new";

    private final String id;
    private final String classPattern;
    private final boolean subtypes;
    private final String method;
    private final List<String> params;
    private final Set<String> at;
    private final List<WhatItem> what;
    private final Limits limits;

    private ProbeSpec(Node n) {
        this.id = n.string("id");
        this.classPattern = n.string("class");
        this.subtypes = n.bool("subtypes");
        this.method = n.string("method");
        List<String> p = n.strings("params");
        this.params = p == null ? null : Collections.unmodifiableList(p);
        this.at = Collections.unmodifiableSet(new LinkedHashSet<>(n.strings("at")));
        List<WhatItem> items = new ArrayList<>();
        for (Node item : n.nodes("what")) {
            items.add(new WhatItem(item.string("key"), item.string("expr")));
        }
        this.what = Collections.unmodifiableList(items);
        this.limits = Limits.from(n.node("limits"));
    }

    static ProbeSpec from(Node n) {
        return new ProbeSpec(n);
    }

    public String id() {
        return id;
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

    public boolean isConstructor() {
        return CONSTRUCTOR.equals(method);
    }

    public List<String> params() {
        return params;
    }

    public Set<String> at() {
        return at;
    }

    public List<WhatItem> what() {
        return what;
    }

    public Limits limits() {
        return limits;
    }
}
