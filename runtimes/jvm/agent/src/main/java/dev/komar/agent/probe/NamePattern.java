package dev.komar.agent.probe;

import java.util.regex.Pattern;

final class NamePattern {

    private final String text;
    private final Pattern regex;

    NamePattern(String text) {
        this.text = text;
        this.regex = text.contains("*") ? compile(text) : null;
    }

    private static Pattern compile(String text) {
        StringBuilder sb = new StringBuilder();
        for (String part : text.split("\\*", -1)) {
            if (sb.length() > 0) {
                sb.append(".*");
            }
            sb.append(Pattern.quote(part));
        }
        return Pattern.compile(sb.toString());
    }

    boolean matches(String name) {
        return regex == null ? text.equals(name) : regex.matcher(name).matches();
    }

    boolean matchesClass(String binaryName) {
        return matches(binaryName) || matches(binaryName.replace('$', '.'));
    }

    @Override
    public String toString() {
        return text;
    }
}
