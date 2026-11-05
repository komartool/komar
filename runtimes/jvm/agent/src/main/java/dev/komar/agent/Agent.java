package dev.komar.agent;

import dev.komar.agent.channel.Channel;
import dev.komar.agent.channel.JsonObject;
import java.io.IOException;
import java.lang.instrument.Instrumentation;
import java.lang.management.ManagementFactory;

public final class Agent {

    private Agent() {
    }

    public static void premain(String args, Instrumentation instrumentation) {
        try {
            start(Session.load(args));
        } catch (IOException | RuntimeException e) {
            return;
        }
    }

    private static void start(Session session) throws IOException {
        Channel channel = Channel.open(session.host(), session.port());
        Runtime.getRuntime().addShutdownHook(new Thread(channel::close, "komar-shutdown"));
        channel.send(new JsonObject()
                .put("kind", "diag")
                .put("level", "info")
                .put("msg", "agent started")
                .put("pid", pid())
                .put("jvm", System.getProperty("java.version"))
                .toString());
    }

    private static long pid() {
        String name = ManagementFactory.getRuntimeMXBean().getName();
        return Long.parseLong(name.substring(0, name.indexOf('@')));
    }
}
