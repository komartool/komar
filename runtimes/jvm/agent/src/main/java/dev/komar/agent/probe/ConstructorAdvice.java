package dev.komar.agent.probe;

import net.bytebuddy.asm.Advice;

public final class ConstructorAdvice {

    private ConstructorAdvice() {
    }

    @Advice.OnMethodEnter(suppress = Throwable.class)
    public static long enter(@SiteId int site, @Advice.AllArguments Object[] args) {
        return Hooks.enter(site, null, args);
    }

    @Advice.OnMethodExit(suppress = Throwable.class)
    public static void exit(@SiteId int site,
                            @Advice.Enter long call,
                            @Advice.This Object self,
                            @Advice.AllArguments Object[] args) {
        Hooks.exit(site, call, self, args, null, null);
    }
}
