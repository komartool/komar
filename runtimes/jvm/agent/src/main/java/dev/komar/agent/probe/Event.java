package dev.komar.agent.probe;

import java.util.Map;

final class Event {

    private final String point;
    private final long call;
    private final long parent;
    private final Object self;
    private final Object[] args;
    private Object ret;
    private boolean hasRet;
    private Throwable thrown;
    private double durationMs = Double.NaN;
    private int line;
    private Map<String, Object> locals;
    private boolean callSite;
    private String callee;
    private Object target;
    private Object[] callArgs;

    Event(String point, long call, long parent, Object self, Object[] args) {
        this.point = point;
        this.call = call;
        this.parent = parent;
        this.self = self;
        this.args = args;
    }

    Event returned(Object value) {
        this.ret = value;
        this.hasRet = true;
        return this;
    }

    Event threw(Throwable error) {
        this.thrown = error;
        return this;
    }

    Event took(double ms) {
        this.durationMs = ms;
        return this;
    }

    Event atLine(int number) {
        this.line = number;
        return this;
    }

    Event withLocals(Map<String, Object> values) {
        this.locals = values;
        return this;
    }

    Event calling(String method, Object receiver, Object[] values) {
        this.callSite = true;
        this.callee = method;
        this.target = receiver;
        this.callArgs = values;
        return this;
    }

    Event next(String nextPoint) {
        return new Event(nextPoint, call, parent, self, args).withLocals(locals).calling(callee, target, callArgs);
    }

    String point() {
        return point;
    }

    long call() {
        return call;
    }

    long parent() {
        return parent;
    }

    Object self() {
        return self;
    }

    Object[] args() {
        return args;
    }

    Object ret() {
        return ret;
    }

    boolean hasRet() {
        return hasRet;
    }

    Throwable thrown() {
        return thrown;
    }

    double durationMs() {
        return durationMs;
    }

    int line() {
        return line;
    }

    Map<String, Object> locals() {
        return locals;
    }

    boolean isCallSite() {
        return callSite;
    }

    String callee() {
        return callee;
    }

    Object target() {
        return target;
    }

    Object[] callArgs() {
        return callArgs;
    }
}
