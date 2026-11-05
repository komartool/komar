package dev.komar.agent.probe;

import java.util.List;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.InsnList;
import org.objectweb.asm.tree.InsnNode;
import org.objectweb.asm.tree.LdcInsnNode;
import org.objectweb.asm.tree.LocalVariableNode;
import org.objectweb.asm.tree.MethodInsnNode;
import org.objectweb.asm.tree.TypeInsnNode;
import org.objectweb.asm.tree.VarInsnNode;

final class HookCode {

    static final int EXTRA_STACK = 10;
    static final String HOOKS = Type.getInternalName(Hooks.class);

    private static final String OBJECT = "java/lang/Object";
    private static final String LINE = "(ILjava/lang/Object;[Ljava/lang/Object;[Ljava/lang/Object;)V";
    private static final String[] BOXES = {
        null, "java/lang/Boolean", "java/lang/Character", "java/lang/Byte", "java/lang/Short",
        "java/lang/Integer", "java/lang/Float", "java/lang/Long", "java/lang/Double",
    };

    private HookCode() {
    }

    static InsnList line(int pointId, Frame frame) {
        InsnList out = prefix(pointId, frame);
        out.add(new MethodInsnNode(Opcodes.INVOKESTATIC, HOOKS, "line", LINE, false));
        return out;
    }

    static InsnList prefix(int pointId, Frame frame) {
        InsnList out = new InsnList();
        out.add(new LdcInsnNode(pointId));
        out.add(frame.thisReady() ? new VarInsnNode(Opcodes.ALOAD, 0) : new InsnNode(Opcodes.ACONST_NULL));
        out.add(array(frame.params(), frame.paramSlots()));
        out.add(locals(frame.locals()));
        return out;
    }

    static InsnList array(Type[] types, int[] slots) {
        InsnList out = new InsnList();
        newArray(out, types.length);
        for (int i = 0; i < types.length; i++) {
            store(out, i, types[i], slots[i]);
        }
        return out;
    }

    static void box(InsnList out, Type type) {
        int sort = type.getSort();
        if (sort >= Type.BOOLEAN && sort <= Type.DOUBLE) {
            String owner = BOXES[sort];
            String desc = "(" + type.getDescriptor() + ")L" + owner + ";";
            out.add(new MethodInsnNode(Opcodes.INVOKESTATIC, owner, "valueOf", desc, false));
        }
    }

    private static InsnList locals(List<LocalVariableNode> locals) {
        InsnList out = new InsnList();
        newArray(out, locals.size());
        for (int i = 0; i < locals.size(); i++) {
            LocalVariableNode v = locals.get(i);
            store(out, i, Type.getType(v.desc), v.index);
        }
        return out;
    }

    private static void newArray(InsnList out, int size) {
        out.add(new LdcInsnNode(size));
        out.add(new TypeInsnNode(Opcodes.ANEWARRAY, OBJECT));
    }

    private static void store(InsnList out, int index, Type type, int slot) {
        out.add(new InsnNode(Opcodes.DUP));
        out.add(new LdcInsnNode(index));
        out.add(new VarInsnNode(type.getOpcode(Opcodes.ILOAD), slot));
        box(out, type);
        out.add(new InsnNode(Opcodes.AASTORE));
    }
}
