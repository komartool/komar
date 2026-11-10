package dev.komar.agent.probe;

import java.util.Arrays;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicLong;

final class Engine {

    private static final double NANOS_PER_MS = 1_000_000.0;

    private final Recorder recorder;
    private final List<ProbeRuntime> probes;
    private final AtomicLong calls = new AtomicLong();
    private final AtomicLong tokens = new AtomicLong();
    private final AtomicBoolean failureReported = new AtomicBoolean();
    private volatile Site[] sites = new Site[0];
    private volatile LinePoint[] lines = new LinePoint[0];
    private volatile CallPoint[] callPoints = new CallPoint[0];

    Engine(Recorder recorder, List<ProbeRuntime> probes) {
        this.recorder = recorder;
        this.probes = probes;
    }

    synchronized int register(Site site) {
        Site[] next = Arrays.copyOf(sites, sites.length + 1);
        next[sites.length] = site;
        sites = next;
        return sites.length - 1;
    }

    synchronized int registerLine(LinePoint point) {
        LinePoint[] next = Arrays.copyOf(lines, lines.length + 1);
        next[lines.length] = point;
        lines = next;
        return lines.length - 1;
    }

    synchronized int registerCall(CallPoint point) {
        CallPoint[] next = Arrays.copyOf(callPoints, callPoints.length + 1);
        next[callPoints.length] = point;
        callPoints = next;
        return callPoints.length - 1;
    }

    void line(int pointId, Object self, Object[] args, Object[] locals) {
        CallStack stack = CallStack.current();
        if (stack.busy()) {
            return;
        }
        LinePoint point = lines[pointId];
        Event e = new Event(Points.LINE, stack.top(), stack.parentOfTop(), self, args)
                .atLine(point.line()).withLocals(point.locals(locals));
        emit(stack, point.site(), e, point.site().probes());
    }

    long callBefore(int pointId, Object self, Object[] args, Object[] locals, Object target, Object[] callArgs) {
        CallStack stack = CallStack.current();
        if (stack.busy()) {
            return 0L;
        }
        CallPoint point = callPoints[pointId];
        List<ProbeRuntime> probes = point.matching(target);
        if (probes.isEmpty()) {
            return 0L;
        }
        Event before = new Event(Points.BEFORE, stack.top(), stack.parentOfTop(), self, args)
                .withLocals(point.locals(locals)).calling(point.callee(), target, callArgs);
        emit(stack, point.site(), before, probes);
        long token = tokens.incrementAndGet();
        stack.pushPending(new Pending(token, before, probes, point, System.nanoTime()));
        return token;
    }

    void finishCall(long token, Object ret, Throwable thrown) {
        CallStack stack = CallStack.current();
        Pending p = stack.popPending(token);
        if (p == null || stack.busy()) {
            return;
        }
        Event e = p.before().next(thrown == null ? Points.AFTER : Points.EXCEPTION);
        if (thrown != null) {
            e.threw(thrown);
        } else if (p.point().returnsValue()) {
            e.returned(ret);
        }
        e.took((System.nanoTime() - p.start()) / NANOS_PER_MS);
        emit(stack, p.point().site(), e, p.probes());
    }

    long enter(int siteId, Object self, Object[] args) {
        CallStack stack = CallStack.current();
        if (stack.busy()) {
            return 0L;
        }
        Site site = sites[siteId];
        long id = calls.incrementAndGet();
        long parent = stack.top();
        if (site.wants(Points.ENTRY)) {
            emit(stack, site, new Event(Points.ENTRY, id, parent, self, args), site.probes());
        }
        stack.push(id, System.nanoTime());
        return id;
    }

    void exit(int siteId, long id, Object self, Object[] args, Object ret, Throwable thrown) {
        CallStack stack = CallStack.current();
        long start = stack.pop(id);
        String point = thrown == null ? Points.EXIT : Points.EXCEPTION;
        Site site = sites[siteId];
        if (stack.busy() || !site.wants(point)) {
            return;
        }
        Event e = new Event(point, id, stack.top(), self, args);
        if (thrown != null) {
            e.threw(thrown);
        } else if (site.returnsValue()) {
            e.returned(ret);
        }
        if (start != CallStack.NOT_FOUND) {
            e.took((System.nanoTime() - start) / NANOS_PER_MS);
        }
        emit(stack, site, e, site.probes());
    }

    void failed(Throwable error) {
        if (failureReported.compareAndSet(false, true)) {
            recorder.diag("error", null, "probe failed inside the agent: " + error);
        }
    }

    void finish() {
        Map<String, Long> whenErrors = new LinkedHashMap<>();
        Map<String, String> unplaced = new LinkedHashMap<>();
        for (ProbeRuntime p : probes) {
            String reason = p.unplacedReason();
            if (reason != null) {
                unplaced.put(p.id(), reason);
            }
            for (String problem : p.bodyProblems()) {
                recorder.diag("warn", p.id(), problem);
            }
            long errors = p.expressions().whenErrors();
            if (errors > 0) {
                whenErrors.put(p.id(), errors);
            }
        }
        recorder.stats(whenErrors, unplaced);
    }

    private void emit(CallStack stack, Site site, Event e, List<ProbeRuntime> probes) {
        stack.busy(true);
        try {
            Scope scope = new Scope(site, e);
            for (ProbeRuntime p : probes) {
                if (p.fires(e) && p.expressions().test(site, scope, recorder) && p.admitHit(recorder)) {
                    recorder.hit(p, site, e, p.expressions().capture(site, scope));
                }
            }
        } finally {
            stack.busy(false);
        }
    }
}
