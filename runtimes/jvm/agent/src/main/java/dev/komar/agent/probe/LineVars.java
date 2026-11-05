package dev.komar.agent.probe;

import java.util.Arrays;
import java.util.List;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import org.objectweb.asm.tree.LocalVariableNode;

final class LineVars {

    private final Recorder recorder;
    private final Set<String> reported = ConcurrentHashMap.newKeySet();

    LineVars(Recorder recorder) {
        this.recorder = recorder;
    }

    void check(ProbeRuntime probe, Site site, int line, String[] locals, List<LocalVariableNode> table) {
        List<String> inScope = Arrays.asList(locals);
        for (String name : probe.expressions().referencedVars()) {
            if (site.hasVar(name) || inScope.contains(name) || declaredElsewhere(name, table)) {
                continue;
            }
            if (reported.add(probe.id() + "\u0000" + name)) {
                String why = table == null || table.isEmpty()
                        ? "local variable names are missing: compiled without -g"
                        : "no variable named " + name + " in " + site.label();
                recorder.diag("warn", probe.id(), "vars." + name + " is not available at line " + line
                        + ": " + why + "; items using it are left out and when is false");
            }
        }
    }

    private static boolean declaredElsewhere(String name, List<LocalVariableNode> table) {
        if (table == null) {
            return false;
        }
        for (LocalVariableNode v : table) {
            if (v.name.equals(name)) {
                return true;
            }
        }
        return false;
    }
}
