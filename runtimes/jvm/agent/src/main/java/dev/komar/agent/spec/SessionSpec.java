package dev.komar.agent.spec;

import dev.komar.agent.json.JsonReader;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

public final class SessionSpec {

    private final String host;
    private final int port;
    private final List<ProbeSpec> probes;

    private SessionSpec(Node n) {
        this.host = n.string("host");
        this.port = n.integer("port");
        List<ProbeSpec> list = new ArrayList<>();
        for (Node p : n.nodes("probes")) {
            list.add(ProbeSpec.from(p));
        }
        this.probes = Collections.unmodifiableList(list);
    }

    public static SessionSpec read(String path) throws IOException {
        if (path == null || path.isEmpty()) {
            throw new IOException("no session file given to the agent");
        }
        String text = new String(Files.readAllBytes(Paths.get(path)), StandardCharsets.UTF_8);
        return new SessionSpec(new Node(JsonReader.parse(text)));
    }

    public String host() {
        return host;
    }

    public int port() {
        return port;
    }

    public List<ProbeSpec> probes() {
        return probes;
    }
}
