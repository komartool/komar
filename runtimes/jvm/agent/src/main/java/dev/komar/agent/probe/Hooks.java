package dev.komar.agent.probe;

public final class Hooks {

    private static volatile Engine engine;

    private Hooks() {
    }

    static void install(Engine e) {
        engine = e;
    }

    @SuppressWarnings("CatchingUnchecked")
    public static long enter(int site, Object self, Object[] args) {
        Engine e = engine;
        if (e == null) {
            return 0L;
        }
        try {
            return e.enter(site, self, args);
        } catch (RuntimeException | LinkageError | StackOverflowError err) {
            return 0L;
        }
    }

    public static void line(int point, Object self, Object[] args, Object[] locals) {
        Engine e = engine;
        if (e == null) {
            return;
        }
        try {
            e.line(point, self, args, locals);
        } catch (RuntimeException | LinkageError | StackOverflowError err) {
            e.failed(err);
        }
    }

    public static long callBefore(int point, Object self, Object[] args, Object[] locals,
                                  Object target, Object[] callArgs) {
        Engine e = engine;
        if (e == null) {
            return 0L;
        }
        try {
            return e.callBefore(point, self, args, locals, target, callArgs);
        } catch (RuntimeException | LinkageError | StackOverflowError err) {
            e.failed(err);
            return 0L;
        }
    }

    public static void callReturned(Object ret, long token) {
        finishCall(token, ret, null);
    }

    public static void callThrew(Throwable thrown, long token) {
        finishCall(token, null, thrown);
    }

    private static void finishCall(long token, Object ret, Throwable thrown) {
        Engine e = engine;
        if (e == null || token == 0L) {
            return;
        }
        try {
            e.finishCall(token, ret, thrown);
        } catch (RuntimeException | LinkageError | StackOverflowError err) {
            e.failed(err);
        }
    }

    public static void exit(int site, long call, Object self, Object[] args, Object ret, Throwable thrown) {
        Engine e = engine;
        if (e == null || call == 0L) {
            return;
        }
        try {
            e.exit(site, call, self, args, ret, thrown);
        } catch (RuntimeException | LinkageError | StackOverflowError err) {
            e.failed(err);
        }
    }
}
