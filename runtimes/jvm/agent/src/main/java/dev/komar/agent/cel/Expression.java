package dev.komar.agent.cel;

import dev.cel.runtime.CelEvaluationException;
import dev.cel.runtime.Program;
import java.util.HashMap;
import java.util.Map;
import java.util.regex.Pattern;

public final class Expression {

    private static final Pattern LOCATION = Pattern.compile("^evaluation error at <input>:\\d+: ");
    private static final Pattern SELECT_ON_VALUE = Pattern.compile(
            "Error resolving field '([^']*)'\\. Field selections must be performed on messages or maps\\.");
    private static final Pattern ARITHMETIC = Pattern.compile("No matching overload for function '_[-+*/%]_'.*");
    private static final String ARITHMETIC_HINT =
            "; int and double do not mix in arithmetic: write 2.0 instead of 2, or wrap a value in double()";

    private final Program program;

    Expression(Program program) {
        this.program = program;
    }

    public Object evaluate(Map<String, Object> roots) throws ExpressionException {
        Map<String, Object> activation = new HashMap<>();
        for (Map.Entry<String, Object> e : roots.entrySet()) {
            activation.put(e.getKey(), CelValues.wrap(e.getValue()));
        }
        try {
            return CelValues.unwrap(program.eval(activation));
        } catch (CelEvaluationException | RuntimeException e) {
            throw new ExpressionException(message(e), e);
        }
    }

    private static String message(Exception e) {
        String msg = e.getMessage() == null ? e.toString() : e.getMessage();
        String plain = LOCATION.matcher(msg).replaceFirst("");
        plain = SELECT_ON_VALUE.matcher(plain).replaceFirst("cannot read field '$1' of null or of a plain value");
        return ARITHMETIC.matcher(plain).matches() ? plain + ARITHMETIC_HINT : plain;
    }
}
