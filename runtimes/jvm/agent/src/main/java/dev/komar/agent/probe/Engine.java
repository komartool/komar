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
    private final AtomicBoolean failureReported = new AtomicBoolean();
    private volatile Site[] sites = new Site[0];
    private volatile LinePoint[] lines = new LinePoint[0];

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

    void line(int pointId, Object self, Object[] args, Object[] locals) {
        CallStack stack = CallStack.current();
        if (stack.busy()) {
            return;
        }
        LinePoint point = lines[pointId];
        Event e = new Event(Points.LINE, stack.top(), stack.parentOfTop(), self, args)
                .atLine(point.line(), point.locals(locals));
        emit(stack, point.site(), e);
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
            emit(stack, site, new Event(Points.ENTRY, id, parent, self, args));
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
        if (thrown == null) {
            e.returned(ret);
        } else {
            e.threw(thrown);
        }
        if (start != CallStack.NOT_FOUND) {
            e.took((System.nanoTime() - start) / NANOS_PER_MS);
        }
        emit(stack, site, e);
    }

    void failed(Throwable error) {
        if (failureReported.compareAndSet(false, true)) {
            recorder.diag("error", null, "probe failed inside the agent: " + error);
        }
    }

    void finish() {
        Map<String, Long> whenErrors = new LinkedHashMap<>();
        for (ProbeRuntime p : probes) {
            String reason = p.unplacedReason();
            if (reason != null) {
                recorder.diag("error", p.id(), reason);
            }
            for (String problem : p.lineProblems()) {
                recorder.diag("warn", p.id(), problem);
            }
            long errors = p.expressions().whenErrors();
            if (errors > 0) {
                whenErrors.put(p.id(), errors);
            }
        }
        recorder.stats(whenErrors);
    }

    private void emit(CallStack stack, Site site, Event e) {
        stack.busy(true);
        try {
            Scope scope = new Scope(site, e);
            for (ProbeRuntime p : site.probes()) {
                if (p.fires(e) && p.expressions().test(site, scope, recorder) && p.admitHit(recorder)) {
                    recorder.hit(p, site, e, p.expressions().capture(site, scope));
                }
            }
        } finally {
            stack.busy(false);
        }
    }
}
