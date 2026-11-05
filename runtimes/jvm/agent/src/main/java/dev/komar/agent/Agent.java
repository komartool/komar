package dev.komar.agent;

import dev.komar.agent.channel.Channel;
import dev.komar.agent.probe.Placement;
import dev.komar.agent.probe.Recorder;
import dev.komar.agent.spec.SessionSpec;
import java.io.IOException;
import java.lang.instrument.Instrumentation;

public final class Agent {

    private Agent() {
    }

    public static void premain(String args, Instrumentation instrumentation) {
        SessionSpec spec;
        Channel channel;
        try {
            spec = SessionSpec.read(args);
            channel = Channel.open(spec.host(), spec.port());
        } catch (IOException | RuntimeException e) {
            return;
        }
        Recorder recorder = new Recorder(channel);
        recorder.diag("info", null, "agent started", "jvm", System.getProperty("java.version"));
        try {
            Placement placement = Placement.install(instrumentation, spec, recorder);
            Runtime.getRuntime().addShutdownHook(new Thread(() -> {
                placement.finish();
                channel.close();
            }, "komar-shutdown"));
        } catch (RuntimeException | LinkageError e) {
            recorder.diag("error", null, "agent failed to place probes: " + e);
            channel.close();
        }
    }
}
