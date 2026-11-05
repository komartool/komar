package dev.komar.agent.probe;

import dev.komar.agent.spec.ProbeSpec;
import net.bytebuddy.description.method.MethodDescription;
import net.bytebuddy.description.type.TypeDescription;

final class Lambdas {

    private static final String PREFIX = "lambda$";
    private static final String INIT = "<init>";

    private Lambdas() {
    }

    static boolean isLambda(String methodName) {
        return methodName.startsWith(PREFIX) && methodName.indexOf('$', PREFIX.length()) > 0;
    }

    static String owner(String methodName) {
        if (!isLambda(methodName)) {
            return INIT.equals(methodName) ? ProbeSpec.CONSTRUCTOR : methodName;
        }
        return methodName.substring(PREFIX.length(), methodName.indexOf('$', PREFIX.length()));
    }

    static MethodDescription.InDefinedShape enclosingMethod(TypeDescription td) {
        try {
            return td.getEnclosingMethod();
        } catch (IllegalStateException e) {
            return null;
        }
    }

    static String enclosingLabel(TypeDescription td, MethodDescription m) {
        MethodDescription.InDefinedShape em = enclosingMethod(td);
        if (em != null) {
            return labelOf(em.getDeclaringType(), em.isConstructor() ? INIT : em.getName());
        }
        return isLambda(m.getName()) ? td.getName() + "." + owner(m.getName()) : null;
    }

    private static String labelOf(TypeDescription owner, String methodName) {
        MethodDescription.InDefinedShape em = enclosingMethod(owner);
        if (em != null) {
            return labelOf(em.getDeclaringType(), em.isConstructor() ? INIT : em.getName());
        }
        return owner.getName() + "." + owner(methodName);
    }
}
