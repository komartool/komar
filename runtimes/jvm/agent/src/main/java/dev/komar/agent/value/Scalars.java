package dev.komar.agent.value;

import java.io.File;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.net.URI;
import java.nio.file.Path;
import java.util.Currency;
import java.util.LinkedHashMap;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;

final class Scalars {

    static final Object NOT_SCALAR = NotScalar.INSTANCE;

    private enum NotScalar { INSTANCE }

    static boolean isScalar(Object rendered) {
        return !(rendered instanceof NotScalar);
    }

    private static final long MAX_SAFE_INTEGER = 1L << 53;
    private static final BigInteger MAX_SAFE = BigInteger.valueOf(MAX_SAFE_INTEGER);
    private static final BigInteger MIN_SAFE = MAX_SAFE.negate();

    private Scalars() {
    }

    static Object render(Object v, int stringLimit) {
        if (v instanceof CharSequence && isJdk(v)) {
            return text(v.toString(), stringLimit);
        }
        Object number = number(v);
        if (isScalar(number)) {
            return number;
        }
        return other(v);
    }

    static boolean isJdk(Object v) {
        return v.getClass().getClassLoader() == null;
    }

    static Object text(String s, int limit) {
        if (s.length() <= limit) {
            return s;
        }
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("@size", (long) s.length());
        m.put("@prefix", s.substring(0, limit));
        return m;
    }

    private static Object number(Object v) {
        if (v instanceof Integer || v instanceof Short || v instanceof Byte) {
            return ((Number) v).longValue();
        }
        if (v instanceof Long) {
            return safeLong((Long) v);
        }
        return floating(v);
    }

    private static Object floating(Object v) {
        if (v instanceof Float) {
            return Double.valueOf(v.toString());
        }
        if (v instanceof Double) {
            return v;
        }
        return bigNumber(v);
    }

    private static Object bigNumber(Object v) {
        if (v instanceof BigInteger && isJdk(v)) {
            BigInteger b = (BigInteger) v;
            boolean safe = b.compareTo(MAX_SAFE) <= 0 && b.compareTo(MIN_SAFE) >= 0;
            return safe ? (Object) b.longValue() : b.toString();
        }
        if (v instanceof BigDecimal && isJdk(v)) {
            return ((BigDecimal) v).toPlainString();
        }
        return atomic(v);
    }

    private static Object atomic(Object v) {
        if (v instanceof AtomicInteger) {
            return (long) ((AtomicInteger) v).get();
        }
        if (v instanceof AtomicLong) {
            return safeLong(((AtomicLong) v).get());
        }
        if (v instanceof AtomicBoolean) {
            return ((AtomicBoolean) v).get();
        }
        return NOT_SCALAR;
    }

    static Object safeLong(long l) {
        return Math.abs(l) <= MAX_SAFE_INTEGER ? (Object) l : Long.toString(l);
    }

    private static Object other(Object v) {
        if (v instanceof Boolean) {
            return v;
        }
        if (v instanceof Character) {
            return v.toString();
        }
        return named(v);
    }

    private static Object named(Object v) {
        if (v instanceof Enum) {
            return ((Enum<?>) v).name();
        }
        if (v instanceof Class) {
            return ((Class<?>) v).getTypeName();
        }
        return isJdk(v) && isPrintedJdkValue(v) ? v.toString() : NOT_SCALAR;
    }

    private static boolean isPrintedJdkValue(Object v) {
        return v.getClass().getName().startsWith("java.time.")
                || v instanceof UUID || v instanceof URI || v instanceof File || v instanceof Path
                || v instanceof Locale || v instanceof Currency;
    }
}
