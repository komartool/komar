package dev.komar.agent.probe;

import java.util.Map;
import net.bytebuddy.asm.AsmVisitorWrapper;
import net.bytebuddy.description.field.FieldDescription;
import net.bytebuddy.description.field.FieldList;
import net.bytebuddy.description.method.MethodList;
import net.bytebuddy.description.type.TypeDescription;
import net.bytebuddy.implementation.Implementation;
import net.bytebuddy.pool.TypePool;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.ClassWriter;

final class BodyWrapper implements AsmVisitorWrapper {

    private final Map<String, MethodPlan> plans;
    private final Engine engine;
    private final BodyVars vars;
    private final boolean hasCalls;

    BodyWrapper(Map<String, MethodPlan> plans, Engine engine, BodyVars vars) {
        this.plans = plans;
        this.engine = engine;
        this.vars = vars;
        boolean calls = false;
        for (MethodPlan plan : plans.values()) {
            calls |= plan.hasCalls();
        }
        this.hasCalls = calls;
    }

    @Override
    public int mergeWriter(int flags) {
        return flags | ClassWriter.COMPUTE_MAXS | (hasCalls ? ClassWriter.COMPUTE_FRAMES : 0);
    }

    @Override
    public int mergeReader(int flags) {
        return flags;
    }

    @Override
    public ClassVisitor wrap(TypeDescription instrumentedType, ClassVisitor classVisitor,
                             Implementation.Context implementationContext, TypePool typePool,
                             FieldList<FieldDescription.InDefinedShape> fields, MethodList<?> methods,
                             int writerFlags, int readerFlags) {
        return new BodyRewriter(classVisitor, plans, engine, vars);
    }
}
