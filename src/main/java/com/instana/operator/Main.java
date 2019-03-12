package com.instana.operator;

import com.sun.net.httpserver.HttpServer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.PrintWriter;
import java.io.StringWriter;
import java.net.InetSocketAddress;
import java.util.Collections;
import java.util.Map;

public class Main {

    private static final Logger logger = LoggerFactory.getLogger(Main.class);
    private static volatile String leaderElectionStatus = "I am not the leader.";
    private static volatile String stackTrace = "";

    // Postpone HTTP framework decision, use com.sun.net.httpserver for now (requires OpenJDK or Oracle JDK).
    public static void main(String[] args) throws Exception {
        runLeaderElectionPollingThread();
        HttpServer httpServer = HttpServer.create(new InetSocketAddress(8080), 10);
        httpServer.createContext("/", httpExchange -> {
            byte[] respBody = getMessage().getBytes("UTF-8");
            httpExchange.getResponseHeaders().put("Context-Type", Collections.singletonList("text/plain; charset=UTF-8"));
            httpExchange.sendResponseHeaders(302, respBody.length);
            httpExchange.getResponseBody().write(respBody);
            httpExchange.getResponseBody().close();
        });
        httpServer.start();
    }

    private static void runLeaderElectionPollingThread() {
        new Thread(() -> {
            try {
                LeaderElector leaderElector = LeaderElector.init();
                leaderElector.waitUntilBecomingLeader();
                leaderElectionStatus = "I am the leader.";
            } catch (Exception e) {
                logger.info("Unexpected Exception in leader election loop: " + e.getMessage(), e);
                stackTrace = stackTraceToString(e);
            }
        }).start();
    }

    private static String getMessage() {
        return getLeaderElectionStatus() + "\n" + getStackTrace() + "\n" + getEnvironmentAsciiTable();
    }

    private static String getLeaderElectionStatus() {
        return "" +
                "LEADER ELECTION STATUS\n" +
                "-----------------------\n" +
                leaderElectionStatus + "\n";
    }

    private static String getStackTrace() {
        if (stackTrace.isEmpty()) {
            return "";
        } else {
            return "" +
                    "EXCEPTION" +
                    "---------" +
                    stackTrace +
                    "\n";
        }
    }

    private static String stackTraceToString(Exception e) {
        StringWriter sw = new StringWriter();
        PrintWriter pw = new PrintWriter(sw);
        e.printStackTrace(pw);
        return sw.toString();
    }

    private static String getEnvironmentAsciiTable() {
        String result = "";
        result = result + "ENVIRONMENT\n";
        result = result + "-----------\n";
        for (Map.Entry<String, String> e : System.getenv().entrySet()) {
            // key length 30 so that KUBERNETES_SERVICE_PORT_HTTPS fits in
            // value length 42 so that the overall table fits in an 80 char terminal window
            result = result + String.format("| %-30s | %-42s |\n", stripAndTruncate(30, e.getKey()), stripAndTruncate(42, e.getValue()));
        }
        return result;
    }

    private static String stripAndTruncate(int length, String s) {
        return truncate(length, stripNewlinesAndTabs(s));
    }

    private static String stripNewlinesAndTabs(String s) {
        if (s == null) {
            return s;
        }
        return s.replaceAll("\\s+", " ");
    }

    private static String truncate(int length, String s) {
        if (s != null && s.length() > length) {
            return s.substring(0, length - 3) + "...";
        }
        return s;
    }
}
