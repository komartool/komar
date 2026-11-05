package dev.komar.agent.probe;

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
}
