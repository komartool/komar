package dev.komar.agent.probe;

import dev.komar.agent.channel.Channel;
import dev.komar.agent.json.JsonWriter;
import java.lang.management.ManagementFactory;
import java.time.Instant;
import java.util.Map;

public final class Recorder {

    private final Channel channel;
    private final long pid;
    private final long startNanos = System.nanoTime();

    public Recorder(Channel channel) {
        this.channel = channel;
        String name = ManagementFactory.getRuntimeMXBean().getName();
        this.pid = Long.parseLong(name.substring(0, name.indexOf('@')));
    }

    public void diag(String level, String probe, String msg, Object... extra) {
        JsonWriter out = new JsonWriter().beginObject()
                .field("kind", "diag")
                .field("level", level);
        if (probe != null) {
            out.field("probe", probe);
        }
        out.field("msg", msg);
        for (int i = 0; i + 1 < extra.length; i += 2) {
            out.field(String.valueOf(extra[i]), extra[i + 1]);
        }
        channel.send(out.field("pid", pid).endObject().toString());
    }

    void stats(Map<String, Long> whenErrors, Map<String, String> unplaced) {
        channel.send(new JsonWriter().beginObject()
                .field("kind", "stats")
                .field("when_errors", whenErrors)
                .field("unplaced", unplaced)
                .endObject().toString());
    }

    void hit(ProbeRuntime probe, Site site, Event e, Map<String, Object> what) {
        Thread thread = Thread.currentThread();
        JsonWriter out = new JsonWriter().beginObject()
                .field("kind", "hit")
                .field("probe", probe.id())
                .field("point", e.point())
                .field("method", site.label());
        if (Points.LINE.equals(e.point())) {
            out.field("line", (long) e.line());
        }
        if (site.enclosing() != null) {
            out.field("enclosing", site.enclosing());
        }
        out.field("call", e.call() == 0L ? null : e.call())
                .field("parent", e.parent() == 0L ? null : e.parent())
                .name("thread").beginObject()
                .field("id", thread.getId())
                .field("name", thread.getName())
                .endObject()
                .field("pid", pid)
                .field("ts", Instant.now().toString())
                .field("t", System.nanoTime() - startNanos)
                .field("what", what);
        channel.send(out.endObject().toString());
    }
}
