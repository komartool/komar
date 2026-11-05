package dev.komar.agent;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.Properties;

final class Session {

    private final String host;
    private final int port;

    private Session(String host, int port) {
        this.host = host;
        this.port = port;
    }

    static Session load(String path) throws IOException {
        if (path == null || path.isEmpty()) {
            throw new IOException("no session file given to the agent");
        }
        Properties props = new Properties();
        try (InputStream in = Files.newInputStream(Paths.get(path))) {
            props.load(in);
        }
        return new Session(required(props, "host"), Integer.parseInt(required(props, "port")));
    }

    private static String required(Properties props, String key) throws IOException {
        String value = props.getProperty(key);
        if (value == null) {
            throw new IOException("session file has no " + key);
        }
        return value;
    }

    String host() {
        return host;
    }

    int port() {
        return port;
    }
}
