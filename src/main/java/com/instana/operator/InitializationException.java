package com.instana.operator;

public class InitializationException extends Exception {

    public InitializationException(String msg) {
        super(msg);
    }

    public InitializationException(String msg, Throwable cause) {
        super(msg, cause);
    }
}
