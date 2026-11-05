package dev.komar.agent.cel;

import dev.cel.common.values.SelectableValue;
import java.util.Map;
import java.util.Optional;

final class LazyMap implements SelectableValue<String> {

    private final Map<?, ?> source;

    LazyMap(Map<?, ?> source) {
        this.source = source;
    }

    Map<?, ?> source() {
        return source;
    }

    @Override
    public Object select(String key) {
        if (!source.containsKey(key)) {
            throw new IllegalArgumentException("no key '" + key + "' in map");
        }
        return CelValues.wrap(source.get(key));
    }

    @Override
    public Optional<Object> find(String key) {
        return source.containsKey(key) ? Optional.of(CelValues.wrap(source.get(key))) : Optional.empty();
    }
}
