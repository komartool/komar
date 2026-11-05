package dev.komar.agent.cel;

import java.lang.reflect.Array;
import java.util.AbstractList;
import java.util.List;
import java.util.RandomAccess;

final class LazyList extends AbstractList<Object> implements RandomAccess {

    private final Object source;

    LazyList(Object source) {
        this.source = source;
    }

    Object source() {
        return source;
    }

    @Override
    public Object get(int index) {
        Object item = source instanceof List ? ((List<?>) source).get(index) : Array.get(source, index);
        return CelValues.wrap(item);
    }

    @Override
    public int size() {
        return source instanceof List ? ((List<?>) source).size() : Array.getLength(source);
    }
}
