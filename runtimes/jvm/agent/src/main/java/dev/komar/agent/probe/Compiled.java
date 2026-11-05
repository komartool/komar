package dev.komar.agent.probe;

import dev.komar.agent.cel.CelEngine;
import dev.komar.agent.cel.Expression;
import dev.komar.agent.cel.ExpressionException;
import dev.komar.agent.spec.WhatItem;
import java.util.Arrays;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.FutureTask;

final class Compiled {

    private static final List<String> ROOTS = Arrays.asList("this", "args", "vars", "ret", "exception", "duration");

    private final WhatItem item;
    private final String root;
    private final FutureTask<Expression> task;

    private Compiled(WhatItem item, String root, FutureTask<Expression> task) {
        this.item = item;
        this.root = root;
        this.task = task;
    }

    static Compiled of(WhatItem item, CelEngine cel) {
        String text = item.expr().trim();
        if (ROOTS.contains(text)) {
            return new Compiled(item, text, null);
        }
        return new Compiled(item, null, new FutureTask<>(() -> cel.compile(text)));
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
        if (task == null) {
            return null;
        }
        task.run();
        try {
            task.get();
            return null;
        } catch (ExecutionException e) {
            return e.getCause().getMessage();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return "interrupted while compiling";
        }
    }

    boolean applies(Scope scope, Site site) {
        return scope.hasAll(item.roots()) && site.hasVars(item.vars());
    }

    Object evaluate(Scope scope) throws ExpressionException {
        if (root != null) {
            return scope.get(root);
        }
        try {
            return task.get().evaluate(scope.roots());
        } catch (ExecutionException e) {
            throw new ExpressionException(e.getCause().getMessage(), e.getCause());
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new ExpressionException("interrupted while compiling", e);
        }
    }
}
