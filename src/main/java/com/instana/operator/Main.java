package com.instana.operator;

import com.sun.net.httpserver.HttpServer;

import java.net.InetSocketAddress;
import java.util.Collections;
import java.util.Map;

public class Main {

    // Postpone framework decision, use OpenJDK's com.sun.net.httpserver for now.
    public static void main(String[] args) throws Exception {
        HttpServer httpServer = HttpServer.create(new InetSocketAddress(8080), 10);
        httpServer.createContext("/", httpExchange -> {
            byte[] respBody = environmentAsciiTable().getBytes("UTF-8");
            httpExchange.getResponseHeaders().put("Context-Type", Collections.singletonList("text/plain; charset=UTF-8"));
            httpExchange.sendResponseHeaders(302, respBody.length);
            httpExchange.getResponseBody().write(respBody);
            httpExchange.getResponseBody().close();
        });
        httpServer.start();
    }

    private static String environmentAsciiTable() {
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
