package dev.komar.agent.probe;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.objectweb.asm.ClassVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.AbstractInsnNode;
import org.objectweb.asm.tree.ClassNode;
import org.objectweb.asm.tree.LineNumberNode;
import org.objectweb.asm.tree.LocalVariableNode;
import org.objectweb.asm.tree.MethodInsnNode;
import org.objectweb.asm.tree.MethodNode;

final class LineRewriter extends ClassNode {

    private static final String INIT = "<init>";

    private final ClassVisitor next;
    private final Map<String, MethodPlan> plans;
    private final Engine engine;
    private final LineVars vars;

    LineRewriter(ClassVisitor next, Map<String, MethodPlan> plans, Engine engine, LineVars vars) {
        super(Opcodes.ASM9);
        this.next = next;
        this.plans = plans;
        this.engine = engine;
        this.vars = vars;
    }

    @Override
    public void visitEnd() {
        for (MethodNode mn : methods) {
            MethodPlan plan = plans.get(mn.name + mn.desc);
            if (plan != null) {
                rewrite(mn, plan);
            }
        }
        accept(next);
    }

    private void rewrite(MethodNode mn, MethodPlan plan) {
        boolean isStatic = (mn.access & Opcodes.ACC_STATIC) != 0;
        int initIndex = INIT.equals(mn.name) ? superCallIndex(mn) : -1;
        List<Placed> placed = new ArrayList<>();
        boolean hasLineTable = false;
        for (AbstractInsnNode n = mn.instructions.getFirst(); n != null; n = n.getNext()) {
            if (!(n instanceof LineNumberNode)) {
                continue;
            }
            hasLineTable = true;
            LineNumberNode ln = (LineNumberNode) n;
            AbstractInsnNode at = firstReal(ln);
            if (at != null && plan.lines().contains(ln.line)) {
                int index = mn.instructions.indexOf(at);
                placed.add(new Placed(ln.line, at, inScope(mn, index, isStatic), !isStatic && index > initIndex));
            }
        }
        if (!hasLineTable) {
            for (ProbeRuntime p : plan.probes()) {
                p.noLineTable();
            }
            return;
        }
        insert(mn, plan, placed, isStatic);
    }

    private void insert(MethodNode mn, MethodPlan plan, List<Placed> placed, boolean isStatic) {
        Type[] params = Type.getArgumentTypes(mn.desc);
        for (Placed p : placed) {
            String[] names = new String[p.locals.size()];
            for (int i = 0; i < names.length; i++) {
                names[i] = p.locals.get(i).name;
            }
            int id = engine.registerLine(new LinePoint(plan.site(), p.line, names));
            mn.instructions.insertBefore(p.at, LineHook.call(id, p.thisReady, params, isStatic ? 0 : 1, p.locals));
            for (ProbeRuntime probe : plan.probesAt(p.line)) {
                probe.placedLine(p.line);
                vars.check(probe, plan.site(), p.line, names, mn.localVariables);
            }
        }
        mn.maxStack += LineHook.EXTRA_STACK;
    }

    private static AbstractInsnNode firstReal(AbstractInsnNode from) {
        for (AbstractInsnNode n = from.getNext(); n != null; n = n.getNext()) {
            if (n instanceof LineNumberNode) {
                return null;
            }
            if (n.getOpcode() >= 0) {
                return n;
            }
        }
        return null;
    }

    private static List<LocalVariableNode> inScope(MethodNode mn, int index, boolean isStatic) {
        List<LocalVariableNode> out = new ArrayList<>();
        if (mn.localVariables == null) {
            return out;
        }
        for (LocalVariableNode v : mn.localVariables) {
            boolean isThis = !isStatic && v.index == 0;
            int start = mn.instructions.indexOf(v.start);
            int end = mn.instructions.indexOf(v.end);
            if (!isThis && start <= index && index < end) {
                out.add(v);
            }
        }
        return out;
    }

    private static int superCallIndex(MethodNode mn) {
        int pending = 0;
        for (AbstractInsnNode n = mn.instructions.getFirst(); n != null; n = n.getNext()) {
            if (n.getOpcode() == Opcodes.NEW) {
                pending++;
            } else if (n.getOpcode() == Opcodes.INVOKESPECIAL && INIT.equals(((MethodInsnNode) n).name)) {
                if (pending == 0) {
                    return mn.instructions.indexOf(n);
                }
                pending--;
            }
        }
        return Integer.MAX_VALUE;
    }

    private static final class Placed {

        private final int line;
        private final AbstractInsnNode at;
        private final List<LocalVariableNode> locals;
        private final boolean thisReady;

        Placed(int line, AbstractInsnNode at, List<LocalVariableNode> locals, boolean thisReady) {
            this.line = line;
            this.at = at;
            this.locals = locals;
            this.thisReady = thisReady;
        }
    }
}
