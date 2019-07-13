package com.instana.operator.client;

import com.instana.operator.FatalErrorHandler;
import io.fabric8.kubernetes.client.Config;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Inject;
import javax.inject.Singleton;

@ApplicationScoped
public class ClientConfigProducer {

  @Inject
  FatalErrorHandler fatalErrorHandler;
  private static final Logger LOGGER = LoggerFactory.getLogger(ClientConfigProducer.class);
  private static final String KUBERNETES_SERVICE_HOST = "KUBERNETES_SERVICE_HOST";
  private static final String KUBERNETES_SERVICE_PORT_HTTPS = "KUBERNETES_SERVICE_PORT_HTTPS";

  @Produces
  @Singleton
  Config makeClientConfig() {
    Config config = Config.autoConfigure(null);
    config.setConnectionTimeout(20_000);
    config.setWebsocketTimeout(20_000);
    config.setRequestTimeout(20_000);
    config.setWatchReconnectLimit(20);
    config.setWebsocketPingInterval(20_000);
    if (!config.getMasterUrl().contains("kubernetes.default.svc")) {
      return config;
    }
    String kubeApiIp = System.getenv(KUBERNETES_SERVICE_HOST);
    if (null == kubeApiIp) {
      LOGGER.error("Environment variable " + KUBERNETES_SERVICE_HOST + " not found. If you are running the operator" +
          " outside of a Kubernetes cluster, make sure that this variable is set to the IP address of the" +
          " Kubernetes API server.");
      fatalErrorHandler.systemExit(-1);
    }
    String kubeApiPort = System.getenv(KUBERNETES_SERVICE_PORT_HTTPS);
    if (null == kubeApiPort) {
      kubeApiPort = "443";
    }
    String masterUrl = "https://" + kubeApiIp + ":" + kubeApiPort + "/";
    config.setMasterUrl(masterUrl);
    return config;
  }
}
