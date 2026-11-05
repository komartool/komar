package dev.komar.agent.spec;

public final class WhatItem {

    private final String key;
    private final String expr;

    WhatItem(String key, String expr) {
        this.key = key;
        this.expr = expr;
    }

    public String key() {
        return key;
    }

    public String expr() {
        return expr;
    }
}
