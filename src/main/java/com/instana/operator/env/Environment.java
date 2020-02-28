package com.instana.operator.env;

@FunctionalInterface
public interface Environment {

  String RELATED_IMAGE_INSTANA_AGENT = "RELATED_IMAGE_INSTANA_AGENT";

  String get(String name);
}
