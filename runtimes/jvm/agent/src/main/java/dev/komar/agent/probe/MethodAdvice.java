package dev.komar.agent.probe;

import net.bytebuddy.asm.Advice;
import net.bytebuddy.implementation.bytecode.assign.Assigner;

public final class MethodAdvice {

    private MethodAdvice() {
    }

    @Advice.OnMethodEnter(suppress = Throwable.class)
    public static long enter(@SiteId int site,
                             @Advice.This(optional = true) Object self,
                             @Advice.AllArguments Object[] args) {
        return Hooks.enter(site, self, args);
    }

    @Advice.OnMethodExit(onThrowable = Throwable.class, suppress = Throwable.class)
    public static void exit(@SiteId int site,
                            @Advice.Enter long call,
                            @Advice.This(optional = true) Object self,
                            @Advice.AllArguments Object[] args,
                            @Advice.Return(typing = Assigner.Typing.DYNAMIC) Object ret,
                            @Advice.Thrown Throwable thrown) {
        Hooks.exit(site, call, self, args, ret, thrown);
    }
}
