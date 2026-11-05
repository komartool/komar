package dev.komar.agent.probe;

import java.util.List;
import org.objectweb.asm.Type;
import org.objectweb.asm.tree.LocalVariableNode;

final class Frame {

    private final boolean thisReady;
    private final Type[] params;
    private final int[] paramSlots;
    private final List<LocalVariableNode> locals;

    Frame(boolean thisReady, Type[] params, int firstSlot, List<LocalVariableNode> locals) {
        this.thisReady = thisReady;
        this.params = params.clone();
        this.paramSlots = new int[params.length];
        int slot = firstSlot;
        for (int i = 0; i < params.length; i++) {
            paramSlots[i] = slot;
            slot += params[i].getSize();
        }
        this.locals = locals;
    }

    boolean thisReady() {
        return thisReady;
    }

    Type[] params() {
        return params.clone();
    }

    int[] paramSlots() {
        return paramSlots.clone();
    }

    List<LocalVariableNode> locals() {
        return locals;
    }

    String[] localNames() {
        String[] names = new String[locals.size()];
        for (int i = 0; i < names.length; i++) {
            names[i] = locals.get(i).name;
        }
        return names;
    }
}
