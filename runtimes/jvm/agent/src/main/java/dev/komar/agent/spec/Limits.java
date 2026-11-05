package dev.komar.agent.spec;

public final class Limits {

    private final int depth;
    private final int items;
    private final int string;
    private final int fields;
    private final int maxHits;
    private final int classes;

    private Limits(Node n) {
        this.depth = n.integer("depth");
        this.items = n.integer("items");
        this.string = n.integer("string");
        this.fields = n.integer("fields");
        this.maxHits = n.integer("max_hits");
        this.classes = n.integer("classes");
    }

    static Limits from(Node n) {
        return new Limits(n);
    }

    public int depth() {
        return depth;
    }

    public int items() {
        return items;
    }

    public int string() {
        return string;
    }

    public int fields() {
        return fields;
    }

    public int maxHits() {
        return maxHits;
    }

    public int classes() {
        return classes;
    }
}
