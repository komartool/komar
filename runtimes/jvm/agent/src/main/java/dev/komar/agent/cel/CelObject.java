package dev.komar.agent.cel;

import dev.cel.common.values.SelectableValue;
import dev.komar.agent.value.Members;
import java.util.Optional;

final class CelObject implements SelectableValue<String> {

    private final Object target;

    CelObject(Object target) {
        this.target = target;
    }

    Object target() {
        return target;
    }

    @Override
    public Object select(String field) {
        return CelValues.wrap(Members.get(target, field));
    }

    @Override
    public Optional<Object> find(String field) {
        if (!Members.has(target, field)) {
            return Optional.empty();
        }
        Object value = Members.get(target, field);
        return value == null ? Optional.empty() : Optional.of(CelValues.wrap(value));
    }

    @Override
    public String toString() {
        return target.getClass().getTypeName();
    }
}
