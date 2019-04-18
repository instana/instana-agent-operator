package com.instana.operator.kubernetes;

public class Label {

  private final String name;
  private final String value;

  public Label(String name, String value) {
    this.name = name;
    this.value = value;
  }

  public String getValue() {
    return value;
  }

  public String getName() {
    return name;
  }
}
