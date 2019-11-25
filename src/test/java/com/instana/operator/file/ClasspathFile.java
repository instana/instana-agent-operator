package com.instana.operator.file;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.stream.Collectors;

public class ClasspathFile {
  public static String read(String resource) throws IOException {
    try (InputStream input = ClasspathFile.class.getResourceAsStream(resource)) {
      try (BufferedReader buffer = new BufferedReader(new InputStreamReader(input))) {
        return buffer.lines().collect(Collectors.joining("\n"));
      }
    }
  }
}
