package dev.komar.agent.value;

import java.lang.reflect.Field;
import java.lang.reflect.Modifier;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Deque;
import java.util.List;

final class Fields {

    private static final ClassValue<Field[]> CACHE = new ClassValue<Field[]>() {
        @Override
        protected Field[] computeValue(Class<?> type) {
            return collect(type);
        }
    };

    private Fields() {
    }

    static Field[] of(Class<?> type) {
        return CACHE.get(type);
    }

    private static Field[] collect(Class<?> type) {
        Deque<Class<?>> chain = new ArrayDeque<>();
        for (Class<?> c = type; c != null && c.getClassLoader() != null; c = c.getSuperclass()) {
            chain.push(c);
        }
        List<Field> out = new ArrayList<>();
        for (Class<?> c : chain) {
            for (Field f : c.getDeclaredFields()) {
                if (!Modifier.isStatic(f.getModifiers()) && !f.isSynthetic()) {
                    out.add(f);
                }
            }
        }
        return out.toArray(new Field[0]);
    }
}
