package com.instana.operator.util;

import io.fabric8.kubernetes.client.Config;

public abstract class ConfigUtils {

  private ConfigUtils() {
  }

  public static Config createClientConfig() throws Exception {
    Config config = Config.autoConfigure(null);
    config.setConnectionTimeout(20_000);
    config.setWebsocketTimeout(20_000);
    config.setRequestTimeout(20_000);
    config.setWatchReconnectLimit(20);
    config.setWebsocketPingInterval(20_000);
    if (!config.getMasterUrl().contains("kubernetes.default.svc")) {
      return config;
    }

    String kubeApiIp = System.getenv("KUBERNETES_SERVICE_HOST");
    if (null == kubeApiIp) {
      throw new Exception("Master url could not be determined");
    }

    String kubeApiPort = System.getenv("KUBERNETES_SERVICE_PORT_HTTPS");
    if (null == kubeApiPort) {
      kubeApiPort = "443";
    }
    String masterUrl = "https://" + kubeApiIp + ":" + kubeApiPort + "/";
    config.setMasterUrl(masterUrl);

    return config;
  }

}
