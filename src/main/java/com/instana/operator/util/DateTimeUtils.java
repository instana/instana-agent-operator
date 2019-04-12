package com.instana.operator.util;

import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;

public class DateTimeUtils {

  public static final DateTimeFormatter UTC = DateTimeFormatter.ISO_ZONED_DATE_TIME;

  public static String nowUTC() {
    ZonedDateTime now = ZonedDateTime.now(ZoneOffset.UTC);
    return UTC.format(now);
  }

}
