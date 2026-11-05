package dev.komar.agent.value;

import dev.komar.agent.json.JsonValue;
import dev.komar.agent.json.JsonWriter;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class ObjectNode implements JsonValue {

    private final String type;
    private final Map<String, Object> fields = new LinkedHashMap<>();
    private final List<String> omitted = new ArrayList<>();
    private boolean cut;
    private int id;

    ObjectNode(String type) {
        this.type = type;
    }

    void put(String name, Object value) {
        fields.put(name, value);
    }

    boolean has(String name) {
        return fields.containsKey(name);
    }

    void omit(String name) {
        omitted.add(name);
    }

    ObjectNode cut() {
        cut = true;
        return this;
    }

    JsonValue reference(Capture capture) {
        if (id == 0) {
            id = capture.nextId();
        }
        return out -> out.beginObject().field("@ref", (long) id).endObject();
    }

    @Override
    public void writeJson(JsonWriter out) {
        out.beginObject().field("@type", type);
        if (id != 0) {
            out.field("@id", (long) id);
        }
        for (Map.Entry<String, Object> e : fields.entrySet()) {
            out.field(e.getKey(), e.getValue());
        }
        if (cut) {
            out.field("@truncated", true);
        } else if (!omitted.isEmpty()) {
            out.field("@truncated", omitted);
        }
        out.endObject();
    }
}
