package dev.komar.agent.cel;

import dev.cel.common.CelAbstractSyntaxTree;
import dev.cel.common.CelOptions;
import dev.cel.common.CelValidationException;
import dev.cel.common.types.SimpleType;
import dev.cel.compiler.CelCompiler;
import dev.cel.compiler.CelCompilerBuilder;
import dev.cel.compiler.CelCompilerFactory;
import dev.cel.parser.CelStandardMacro;
import dev.cel.runtime.CelEvaluationException;
import dev.cel.runtime.CelRuntime;
import dev.cel.runtime.CelRuntimeFactory;

public final class CelEngine {

    private static final String[] ROOTS = {"this", "args", "vars", "ret", "exception", "duration", "call"};

    private CelCompiler compiler;
    private CelRuntime runtime;

    private synchronized void init() {
        if (compiler != null) {
            return;
        }
        CelOptions options = CelOptions.current().enableHeterogeneousNumericComparisons(true).build();
        CelCompilerBuilder builder = CelCompilerFactory.standardCelCompilerBuilder()
                .setOptions(options)
                .setStandardMacros(CelStandardMacro.STANDARD_MACROS);
        for (String root : ROOTS) {
            builder.addVar(root, SimpleType.DYN);
        }
        this.compiler = builder.build();
        this.runtime = CelRuntimeFactory.plannerRuntimeBuilder().setOptions(options).build();
    }

    public Expression compile(String text) throws ExpressionException {
        try {
            init();
            CelAbstractSyntaxTree ast = compiler.compile(text).getAst();
            return new Expression(runtime.createProgram(ast));
        } catch (CelValidationException | CelEvaluationException | RuntimeException e) {
            throw new ExpressionException(e.getMessage(), e);
        }
    }
}
