package dev.komar.agent.spec;

import java.util.Collections;
import java.util.List;

public final class WhatItem {

    private final String key;
    private final String expr;
    private final List<String> roots;
    private final List<String> vars;
    private final List<String> callFields;

    private WhatItem(Node n) {
        this.key = n.string("key");
        this.expr = n.string("expr");
        this.roots = listOrEmpty(n.strings("roots"));
        this.vars = listOrEmpty(n.strings("vars"));
        this.callFields = listOrEmpty(n.strings("call_fields"));
    }

    static WhatItem from(Node n) {
        return new WhatItem(n);
    }

    private static List<String> listOrEmpty(List<String> list) {
        return list == null ? Collections.<String>emptyList() : Collections.unmodifiableList(list);
    }

    public String key() {
        return key;
    }

    public String expr() {
        return expr;
    }

    public List<String> roots() {
        return roots;
    }

    public List<String> vars() {
        return vars;
    }

    public List<String> callFields() {
        return callFields;
    }
}
