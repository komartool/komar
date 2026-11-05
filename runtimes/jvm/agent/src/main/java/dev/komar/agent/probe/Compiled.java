package dev.komar.agent.probe;

import dev.komar.agent.cel.CelEngine;
import dev.komar.agent.cel.Expression;
import dev.komar.agent.cel.ExpressionException;
import dev.komar.agent.spec.WhatItem;
import java.util.Arrays;
import java.util.List;

final class Compiled {

    private static final List<String> ROOTS = Arrays.asList("this", "args", "vars", "ret", "exception", "duration");

    private final WhatItem item;
    private final String root;
    private final Expression expression;
    private final String error;

    private Compiled(WhatItem item, String root, Expression expression, String error) {
        this.item = item;
        this.root = root;
        this.expression = expression;
        this.error = error;
    }

    static Compiled of(WhatItem item, CelEngine cel) {
        String text = item.expr().trim();
        if (ROOTS.contains(text)) {
            return new Compiled(item, text, null, null);
        }
        try {
            return new Compiled(item, null, cel.compile(text), null);
        } catch (ExpressionException e) {
            return new Compiled(item, null, null, e.getMessage());
        }
    }

    String key() {
        return item.key();
    }

    List<String> vars() {
        return item.vars();
    }

    boolean isRoot(String name) {
        return name.equals(root);
    }

    String error() {
        return error;
    }

    boolean applies(Scope scope, Site site) {
        return scope.hasAll(item.roots()) && site.hasVars(item.vars());
    }

    Object evaluate(Scope scope) throws ExpressionException {
        return root != null ? scope.get(root) : expression.evaluate(scope.roots());
    }
}
