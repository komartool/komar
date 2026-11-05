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

final class LineWrapper implements AsmVisitorWrapper {

    private final Map<String, MethodPlan> plans;
    private final Engine engine;
    private final LineVars vars;

    LineWrapper(Map<String, MethodPlan> plans, Engine engine, LineVars vars) {
        this.plans = plans;
        this.engine = engine;
        this.vars = vars;
    }

    @Override
    public int mergeWriter(int flags) {
        return flags | ClassWriter.COMPUTE_MAXS;
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
        return new LineRewriter(classVisitor, plans, engine, vars);
    }
}
