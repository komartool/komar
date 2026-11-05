package dev.komar.agent.probe;

import java.time.Instant;

final class Clock {

    private volatile Stamp last = new Stamp(-1L, "");

    String now() {
        long millis = System.currentTimeMillis();
        Stamp s = last;
        if (s.millis == millis) {
            return s.text;
        }
        Stamp fresh = new Stamp(millis, Instant.ofEpochMilli(millis).toString());
        last = fresh;
        return fresh.text;
    }

    private static final class Stamp {

        private final long millis;
        private final String text;

        Stamp(long millis, String text) {
            this.millis = millis;
            this.text = text;
        }
    }
}
