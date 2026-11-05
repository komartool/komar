package dev.komar.agent.probe;

import java.util.Arrays;

final class CallStack {

    static final long NOT_FOUND = -1L;

    private static final int INITIAL_CAPACITY = 16;
    private static final ThreadLocal<CallStack> CURRENT = ThreadLocal.withInitial(CallStack::new);

    private long[] ids = new long[INITIAL_CAPACITY];
    private long[] starts = new long[INITIAL_CAPACITY];
    private int size;
    private boolean busy;

    static CallStack current() {
        return CURRENT.get();
    }

    long top() {
        return size == 0 ? 0L : ids[size - 1];
    }

    long parentOfTop() {
        return size < 2 ? 0L : ids[size - 2];
    }

    void push(long id, long start) {
        if (size == ids.length) {
            ids = Arrays.copyOf(ids, size * 2);
            starts = Arrays.copyOf(starts, size * 2);
        }
        ids[size] = id;
        starts[size] = start;
        size++;
    }

    long pop(long id) {
        for (int i = size - 1; i >= 0; i--) {
            if (ids[i] == id) {
                size = i;
                return starts[i];
            }
        }
        return NOT_FOUND;
    }

    boolean busy() {
        return busy;
    }

    void busy(boolean value) {
        busy = value;
    }
}
