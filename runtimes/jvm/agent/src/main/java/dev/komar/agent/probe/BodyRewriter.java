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

final class BodyRewriter extends ClassNode {

    private static final String INIT = "<init>";
    private static final String AGENT_PACKAGE = "dev/komar/";

    private final ClassVisitor next;
    private final Map<String, MethodPlan> plans;
    private final Engine engine;
    private final BodyVars vars;

    BodyRewriter(ClassVisitor next, Map<String, MethodPlan> plans, Engine engine, BodyVars vars) {
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
        int initIndex = INIT.equals(mn.name) ? superCallIndex(mn) : -1;
        if (plan.hasCalls()) {
            wrapCalls(mn, plan, initIndex);
        }
        if (!plan.lineProbes().isEmpty()) {
            insertLines(mn, plan, initIndex);
        }
    }

    private void wrapCalls(MethodNode mn, MethodPlan plan, int initIndex) {
        List<MethodInsnNode> calls = new ArrayList<>();
        for (AbstractInsnNode n = mn.instructions.getFirst(); n != null; n = n.getNext()) {
            if (n instanceof MethodInsnNode && isWrappable((MethodInsnNode) n)
                    && mn.instructions.indexOf(n) > initIndex) {
                calls.add((MethodInsnNode) n);
            }
        }
        for (MethodInsnNode insn : calls) {
            List<ProbeRuntime> matched = plan.probesCalling(insn);
            if (matched.isEmpty()) {
                continue;
            }
            Frame frame = frame(mn, mn.instructions.indexOf(insn), initIndex);
            boolean returnsValue = Type.getReturnType(insn.desc).getSort() != Type.VOID;
            boolean isStatic = insn.getOpcode() == Opcodes.INVOKESTATIC;
            CallPoint point = new CallPoint(plan.site(), frame.localNames(), matched, returnsValue, isStatic);
            int id = engine.registerCall(point);
            CallSiteCode.wrap(mn, insn, id, frame);
            for (ProbeRuntime p : matched) {
                p.placedCall();
                vars.check(p, plan.site(), "the call to " + insn.name, frame.localNames(), mn.localVariables);
            }
        }
    }

    private static boolean isWrappable(MethodInsnNode insn) {
        return !INIT.equals(insn.name) && !insn.owner.startsWith(AGENT_PACKAGE);
    }

    private void insertLines(MethodNode mn, MethodPlan plan, int initIndex) {
        boolean hasLineTable = false;
        List<LineNumberNode> targets = new ArrayList<>();
        for (AbstractInsnNode n = mn.instructions.getFirst(); n != null; n = n.getNext()) {
            if (n instanceof LineNumberNode) {
                hasLineTable = true;
                if (plan.lines().contains(((LineNumberNode) n).line) && firstReal(n) != null) {
                    targets.add((LineNumberNode) n);
                }
            }
        }
        if (!hasLineTable) {
            for (ProbeRuntime p : plan.lineProbes()) {
                p.noLineTable();
            }
            return;
        }
        for (LineNumberNode ln : targets) {
            AbstractInsnNode at = firstReal(ln);
            Frame frame = frame(mn, mn.instructions.indexOf(at), initIndex);
            int id = engine.registerLine(new LinePoint(plan.site(), ln.line, frame.localNames()));
            mn.instructions.insertBefore(at, HookCode.line(id, frame));
            for (ProbeRuntime probe : plan.probesAt(ln.line)) {
                probe.placedLine(ln.line);
                vars.check(probe, plan.site(), "line " + ln.line, frame.localNames(), mn.localVariables);
            }
        }
        mn.maxStack += HookCode.EXTRA_STACK;
    }

    private static Frame frame(MethodNode mn, int index, int initIndex) {
        boolean isStatic = (mn.access & Opcodes.ACC_STATIC) != 0;
        return new Frame(!isStatic && index > initIndex, Type.getArgumentTypes(mn.desc), isStatic ? 0 : 1,
                inScope(mn, index, isStatic));
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
}
