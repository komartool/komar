package dev.komar.agent.probe;

import java.util.List;

final class Pending {

    private final long token;
    private final Event before;
    private final List<ProbeRuntime> probes;
    private final CallPoint point;
    private final long start;

    Pending(long token, Event before, List<ProbeRuntime> probes, CallPoint point, long start) {
        this.token = token;
        this.before = before;
        this.probes = probes;
        this.point = point;
        this.start = start;
    }

    long token() {
        return token;
    }

    Event before() {
        return before;
    }

    List<ProbeRuntime> probes() {
        return probes;
    }

    CallPoint point() {
        return point;
    }

    long start() {
        return start;
    }
}
