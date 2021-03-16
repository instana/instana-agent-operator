/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.env;

import java.util.Map;

public interface Environment {

  String RELATED_IMAGE_INSTANA_AGENT = "RELATED_IMAGE_INSTANA_AGENT";
  String RELATED_IMAGE_PULLPOLICY_INSTANA_AGENT = "RELATED_IMAGE_PULLPOLICY_INSTANA_AGENT";
  String RELATED_INSTANA_OTEL_ACTIVE = "RELATED_INSTANA_OTEL_ACTIVE";
  String OPERATOR_NAMESPACE = "OPERATOR_NAMESPACE";
  String TARGET_NAMESPACES = "TARGET_NAMESPACES";
  String POD_NAME = "POD_NAME";
  String KUBERNETES_SERVICE_HOST = "KUBERNETES_SERVICE_HOST";
  String KUBERNETES_SERVICE_PORT_HTTPS = "KUBERNETES_SERVICE_PORT_HTTPS";

  String get(String name);

  Map<String, String> all();

  static Environment fromMap(Map<String, String> map) {
    return new Environment() {
      @Override
      public String get(String name) {
        return map.get(name);
      }

      @Override
      public Map<String, String> all() {
        return map;
      }
    };
  }
}
