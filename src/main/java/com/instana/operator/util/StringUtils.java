package com.instana.operator.util;

public class StringUtils {

  // from apache commons-lang
  public static boolean isBlank(final CharSequence cs) {
    int strLen;
    if (cs == null || (strLen = cs.length()) == 0) {
      return true;
    }
    for (int i = 0; i < strLen; i++) {
      if (!Character.isWhitespace(cs.charAt(i))) {
        return false;
      }
    }
    return true;
  }

  // from apache commons-lang
  public static boolean isEmpty(final CharSequence cs) {
    return cs == null || cs.length() == 0;
  }
}
