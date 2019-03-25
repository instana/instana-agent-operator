package com.instana.operator.util;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;

import com.instana.operator.InitializationException;

public class NamespaceUtils {

  private static String findNamespace() throws InitializationException {
    try {
      byte[] bytes = Files.readAllBytes(Paths.get("/var/run/secrets/kubernetes.io/serviceaccount/namespace"));
      return new String(bytes, StandardCharsets.UTF_8).trim();
    } catch (IOException e) {
      throw new InitializationException(
          "Namespace not found. This container seems to be running outside of a Kubernetes cluster.");
    }
  }

}
