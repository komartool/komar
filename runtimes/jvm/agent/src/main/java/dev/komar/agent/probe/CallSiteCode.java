package dev.komar.agent.probe;

import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.InsnList;
import org.objectweb.asm.tree.InsnNode;
import org.objectweb.asm.tree.JumpInsnNode;
import org.objectweb.asm.tree.LabelNode;
import org.objectweb.asm.tree.MethodInsnNode;
import org.objectweb.asm.tree.MethodNode;
import org.objectweb.asm.tree.TryCatchBlockNode;
import org.objectweb.asm.tree.VarInsnNode;

final class CallSiteCode {

    private static final String BEFORE =
            "(ILjava/lang/Object;[Ljava/lang/Object;[Ljava/lang/Object;Ljava/lang/Object;[Ljava/lang/Object;)J";
    private static final String RETURNED = "(Ljava/lang/Object;J)V";
    private static final String THREW = "(Ljava/lang/Throwable;J)V";
    private static final String THROWABLE = "java/lang/Throwable";
    private static final int LONG_SIZE = 2;

    private CallSiteCode() {
    }

    static void wrap(MethodNode mn, MethodInsnNode insn, int pointId, Frame frame) {
        Layout layout = new Layout(mn, insn);
        LabelNode start = new LabelNode();
        mn.instructions.insertBefore(insn, before(pointId, frame, layout, start));
        LabelNode end = new LabelNode();
        LabelNode handler = new LabelNode();
        mn.instructions.insert(insn, after(Type.getReturnType(insn.desc), layout.token, end, handler));
        mn.tryCatchBlocks.add(0, new TryCatchBlockNode(start, end, handler, THROWABLE));
    }

    private static InsnList before(int pointId, Frame frame, Layout layout, LabelNode start) {
        Type[] argTypes = layout.argTypes;
        int[] slots = layout.slots;
        int receiver = layout.receiver;
        int token = layout.token;
        InsnList out = new InsnList();
        for (int i = argTypes.length - 1; i >= 0; i--) {
            out.add(new VarInsnNode(argTypes[i].getOpcode(Opcodes.ISTORE), slots[i]));
        }
        if (receiver >= 0) {
            out.add(new VarInsnNode(Opcodes.ASTORE, receiver));
        }
        out.add(HookCode.prefix(pointId, frame));
        out.add(receiver >= 0 ? new VarInsnNode(Opcodes.ALOAD, receiver) : new InsnNode(Opcodes.ACONST_NULL));
        out.add(HookCode.array(argTypes, slots));
        out.add(new MethodInsnNode(Opcodes.INVOKESTATIC, HookCode.HOOKS, "callBefore", BEFORE, false));
        out.add(new VarInsnNode(Opcodes.LSTORE, token));
        if (receiver >= 0) {
            out.add(new VarInsnNode(Opcodes.ALOAD, receiver));
        }
        for (int i = 0; i < argTypes.length; i++) {
            out.add(new VarInsnNode(argTypes[i].getOpcode(Opcodes.ILOAD), slots[i]));
        }
        out.add(start);
        return out;
    }

    private static final class Layout {

        private final Type[] argTypes;
        private final int[] slots;
        private final int receiver;
        private final int token;

        Layout(MethodNode mn, MethodInsnNode insn) {
            this.argTypes = Type.getArgumentTypes(insn.desc);
            this.slots = new int[argTypes.length];
            int next = mn.maxLocals;
            for (int i = 0; i < argTypes.length; i++) {
                slots[i] = next;
                next += argTypes[i].getSize();
            }
            this.receiver = insn.getOpcode() == Opcodes.INVOKESTATIC ? -1 : next++;
            this.token = next;
            mn.maxLocals = next + LONG_SIZE;
        }
    }

    private static InsnList after(Type ret, int token, LabelNode end, LabelNode handler) {
        InsnList out = new InsnList();
        LabelNode done = new LabelNode();
        out.add(end);
        if (ret.getSort() == Type.VOID) {
            out.add(new InsnNode(Opcodes.ACONST_NULL));
        } else {
            out.add(new InsnNode(ret.getSize() == LONG_SIZE ? Opcodes.DUP2 : Opcodes.DUP));
            HookCode.box(out, ret);
        }
        out.add(new VarInsnNode(Opcodes.LLOAD, token));
        out.add(new MethodInsnNode(Opcodes.INVOKESTATIC, HookCode.HOOKS, "callReturned", RETURNED, false));
        out.add(new JumpInsnNode(Opcodes.GOTO, done));
        out.add(handler);
        out.add(new InsnNode(Opcodes.DUP));
        out.add(new VarInsnNode(Opcodes.LLOAD, token));
        out.add(new MethodInsnNode(Opcodes.INVOKESTATIC, HookCode.HOOKS, "callThrew", THREW, false));
        out.add(new InsnNode(Opcodes.ATHROW));
        out.add(done);
        return out;
    }
}
