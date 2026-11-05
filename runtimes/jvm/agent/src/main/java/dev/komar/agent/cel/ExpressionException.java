package dev.komar.agent.cel;

public final class ExpressionException extends Exception {

    private static final long serialVersionUID = 1L;

    ExpressionException(String message, Throwable cause) {
        super(message, cause);
    }
}
