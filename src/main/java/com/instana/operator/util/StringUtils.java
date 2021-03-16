/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
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

  public static boolean getBoolean(final String booleanValue) {
    try{
      return Boolean.parseBoolean(booleanValue);
    }catch (Exception es){
      // Boolean not parsable
    }
    return false;
  }

  public static boolean isInteger(String intValue) {
    try {
      Integer.parseInt(intValue);
    } catch(NumberFormatException e) {
      return false;
    } catch(NullPointerException e) {
      return false;
    }
    return true;
  }

  public static Integer getInteger(String intValue) {
    try {
      return Integer.parseInt(intValue);
    } catch(NumberFormatException e) {

    } catch(NullPointerException e) {
    }
    return null;
  }

  // from apache commons-lang
  public static boolean isEmpty(final CharSequence cs) {
    return cs == null || cs.length() == 0;
  }
}
