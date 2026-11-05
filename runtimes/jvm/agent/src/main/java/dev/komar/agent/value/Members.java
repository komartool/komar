package dev.komar.agent.value;

import java.lang.reflect.Field;

public final class Members {

    private static final String MESSAGE = "message";
    private static final String CAUSE = "cause";

    private Members() {
    }

    public static boolean has(Object target, String name) {
        return isThrowableMember(target, name) || find(target.getClass(), name) != null;
    }

    public static Object get(Object target, String name) {
        if (isThrowableMember(target, name)) {
            Throwable t = (Throwable) target;
            return MESSAGE.equals(name) ? t.getMessage() : t.getCause();
        }
        Field f = find(target.getClass(), name);
        if (f == null) {
            throw new IllegalArgumentException("no field '" + name + "' in " + target.getClass().getTypeName());
        }
        try {
            f.setAccessible(true);
            return f.get(target);
        } catch (IllegalAccessException | RuntimeException e) {
            throw new IllegalStateException("cannot read field '" + name + "' of "
                    + target.getClass().getTypeName() + ": " + e, e);
        }
    }

    private static boolean isThrowableMember(Object target, String name) {
        if (!(target instanceof Throwable)) {
            return false;
        }
        if (MESSAGE.equals(name)) {
            return Throwables.isJdkMethod((Throwable) target, "getMessage");
        }
        return CAUSE.equals(name) && Throwables.isJdkMethod((Throwable) target, "getCause");
    }

    private static Field find(Class<?> type, String name) {
        Field found = null;
        for (Field f : Fields.of(type)) {
            if (f.getName().equals(name)) {
                found = f;
            }
        }
        return found;
    }
}
