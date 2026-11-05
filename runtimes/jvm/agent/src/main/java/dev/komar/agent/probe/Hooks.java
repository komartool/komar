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
