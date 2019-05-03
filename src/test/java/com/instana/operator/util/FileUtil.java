package com.instana.operator.util;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.stream.Collectors;

public class FileUtil {
  public static String readFromClasspath(String resource) throws IOException {
    try (InputStream input = FileUtil.class.getResourceAsStream(resource)) {
      try (BufferedReader buffer = new BufferedReader(new InputStreamReader(input))) {
        return buffer.lines().collect(Collectors.joining("\n"));
      }
    }
  }
}
