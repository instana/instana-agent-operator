/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.client;

import com.instana.operator.FatalErrorHandler;
import com.instana.operator.env.Environment;
import io.fabric8.kubernetes.client.Config;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Inject;
import javax.inject.Singleton;

import static com.instana.operator.env.Environment.KUBERNETES_SERVICE_HOST;
import static com.instana.operator.env.Environment.KUBERNETES_SERVICE_PORT_HTTPS;

@ApplicationScoped
public class ClientConfigProducer {

  @Inject
  FatalErrorHandler fatalErrorHandler;
  private static final Logger LOGGER = LoggerFactory.getLogger(ClientConfigProducer.class);

  @Inject
  Environment environment;

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
    String kubeApiIp = environment.get(KUBERNETES_SERVICE_HOST);
    if (null == kubeApiIp) {
      LOGGER.error("Environment variable " + KUBERNETES_SERVICE_HOST + " not found. If you are running the operator" +
          " outside of a Kubernetes cluster, make sure that this variable is set to the IP address of the" +
          " Kubernetes API server.");
      fatalErrorHandler.systemExit(-1);
    }
    String kubeApiPort = environment.get(KUBERNETES_SERVICE_PORT_HTTPS);
    if (null == kubeApiPort) {
      kubeApiPort = "443";
    }
    String masterUrl = "https://" + kubeApiIp + ":" + kubeApiPort + "/";
    config.setMasterUrl(masterUrl);
    return config;
  }
}
