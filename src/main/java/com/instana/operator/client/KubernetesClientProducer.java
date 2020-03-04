package com.instana.operator.client;

import com.instana.operator.FatalErrorHandler;
import com.instana.operator.customresource.DoneableInstanaAgent;
import com.instana.operator.customresource.InstanaAgent;
import com.instana.operator.customresource.InstanaAgentList;
import io.fabric8.kubernetes.api.model.apiextensions.CustomResourceDefinition;
import io.fabric8.kubernetes.client.Config;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.dsl.MixedOperation;
import io.fabric8.kubernetes.client.dsl.Resource;
import io.fabric8.kubernetes.client.internal.SSLUtils;
import io.fabric8.kubernetes.client.utils.BackwardsCompatibilityInterceptor;
import io.fabric8.kubernetes.client.utils.ImpersonatorInterceptor;
import io.fabric8.kubernetes.client.utils.Utils;
import io.fabric8.kubernetes.internal.KubernetesDeserializer;
import okhttp3.ConnectionSpec;
import okhttp3.Credentials;
import okhttp3.Dispatcher;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.logging.HttpLoggingInterceptor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.enterprise.inject.Produces;
import javax.inject.Inject;
import javax.inject.Named;
import javax.inject.Singleton;
import javax.net.ssl.KeyManager;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;
import java.net.Proxy;
import java.util.Arrays;
import java.util.Optional;
import java.util.concurrent.TimeUnit;

import static com.instana.operator.env.Environment.OPERATOR_NAMESPACE;
import static com.instana.operator.util.StringUtils.isEmpty;
import static okhttp3.ConnectionSpec.CLEARTEXT;

@ApplicationScoped
public class KubernetesClientProducer {

  public static final String KUBERNETES_HTTP_CLIENT = "KUBERNETES_HTTP_CLIENT";
  public static final String AGENT_POD_HTTP_CLIENT = "AGENT_POD_CLIENT";

  private static final Logger LOGGER = LoggerFactory.getLogger(KubernetesClientProducer.class);
  public static final String CRD_GROUP = "instana.io";
  public static final String CRD_NAME = "agents." + CRD_GROUP;
  public static final String CRD_VERSION = "v1beta1";
  public static final String CR_KIND = InstanaAgent.class.getSimpleName();

  @Inject
  FatalErrorHandler fatalErrorHandler;

  @Produces
  @Singleton
  @Named(AGENT_POD_HTTP_CLIENT)
  OkHttpClient makeAgentPodHttpClient() {
    return new OkHttpClient();
  }

  @Produces
  @Singleton
  DefaultKubernetesClient makeDefaultClient(Config config, @Named(KUBERNETES_HTTP_CLIENT) OkHttpClient httpClient) {
    return new DefaultKubernetesClient(httpClient, config);
  }

  @Produces
  @Singleton
  MixedOperation<InstanaAgent, InstanaAgentList, DoneableInstanaAgent, Resource<InstanaAgent, DoneableInstanaAgent>>
  makeCustomResourceClient(DefaultKubernetesClient defaultClient, @Named(OPERATOR_NAMESPACE) String operatorNamespace) {
    Optional<CustomResourceDefinition> crd = defaultClient
        .inNamespace(operatorNamespace)
        .customResourceDefinitions().list().getItems().stream()
        .filter(c -> CRD_NAME.equals(c.getMetadata().getName()))
        .findFirst();
    if (!crd.isPresent()) {
      LOGGER.error(
          "Custom resource definition " + CRD_NAME + " not found. Please create the CRD using the provided YAML.");
      fatalErrorHandler.systemExit(-1);
    }
    KubernetesDeserializer.registerCustomKind(CRD_GROUP + "/" + CRD_VERSION, CR_KIND, InstanaAgent.class);
    return defaultClient
        .customResources(crd.get(), InstanaAgent.class, InstanaAgentList.class, DoneableInstanaAgent.class);
  }

  @Produces
  @Singleton
  @Named(KUBERNETES_HTTP_CLIENT)
  OkHttpClient makeKubernetesHttpClient(Config config) {
    try {
      OkHttpClient.Builder httpClientBuilder = new OkHttpClient.Builder();

      // Follow any redirects
      httpClientBuilder.followRedirects(true);
      httpClientBuilder.followSslRedirects(true);

      // Set to trust all certificates by default, unless customer has specified own certificate / store to be used
      if (config.getCaCertFile() == null && config.getCaCertData() == null &&
          config.getClientCertFile() == null && config.getClientCertData() == null &&
          config.getClientKeyFile() == null && config.getClientKeyData() == null &&
          config.getKeyStoreFile() == null) {

        config.setTrustCerts(true);
      }

      if (config.isTrustCerts() || config.isDisableHostnameVerification()) {
        httpClientBuilder.hostnameVerifier((s, sslSession) -> true);
      }

      TrustManager[] trustManagers = SSLUtils.trustManagers(config);
      KeyManager[] keyManagers = SSLUtils.keyManagers(config);

      if (keyManagers != null || trustManagers != null || config.isTrustCerts()) {
        X509TrustManager trustManager = null;
        if (trustManagers != null && trustManagers.length == 1) {
          trustManager = (X509TrustManager) trustManagers[0];
        }

        SSLContext sslContext = SSLUtils.sslContext(keyManagers, trustManagers);
        httpClientBuilder.sslSocketFactory(sslContext.getSocketFactory(), trustManager);
      } else {
        SSLContext context = SSLContext.getInstance("TLSv1.2");
        context.init(keyManagers, trustManagers, null);
        httpClientBuilder.sslSocketFactory(context.getSocketFactory(), (X509TrustManager) trustManagers[0]);
      }

      httpClientBuilder
          .addInterceptor(chain -> {
            Request request = chain.request();
            if (!isEmpty(config.getUsername()) && !isEmpty(config.getPassword())) {
              Request authReq = chain.request().newBuilder()
                  .addHeader("Authorization", Credentials.basic(config.getUsername(), config.getPassword())).build();
              return chain.proceed(authReq);
            } else if (Utils.isNotNullOrEmpty(config.getOauthToken())) {
              Request authReq = chain.request().newBuilder()
                  .addHeader("Authorization", "Bearer " + config.getOauthToken()).build();
              return chain.proceed(authReq);
            }
            return chain.proceed(request);
          })
          .addInterceptor(new ImpersonatorInterceptor(config))
          .addInterceptor(new BackwardsCompatibilityInterceptor());

      Logger reqLogger = LoggerFactory.getLogger(HttpLoggingInterceptor.class);
      if (reqLogger.isTraceEnabled()) {
        HttpLoggingInterceptor loggingInterceptor = new HttpLoggingInterceptor();
        loggingInterceptor.setLevel(HttpLoggingInterceptor.Level.BODY);
        httpClientBuilder.addNetworkInterceptor(loggingInterceptor);
      }

      if (config.getConnectionTimeout() > 0) {
        httpClientBuilder.connectTimeout(config.getConnectionTimeout(), TimeUnit.MILLISECONDS);
      }

      if (config.getRequestTimeout() > 0) {
        httpClientBuilder.readTimeout(config.getRequestTimeout(), TimeUnit.MILLISECONDS);
      }

      if (config.getWebsocketPingInterval() > 0) {
        httpClientBuilder.pingInterval(config.getWebsocketPingInterval(), TimeUnit.MILLISECONDS);
      }

      if (config.getMaxConcurrentRequestsPerHost() > 0) {
        Dispatcher dispatcher = new Dispatcher();
        dispatcher.setMaxRequests(config.getMaxConcurrentRequests());
        dispatcher.setMaxRequestsPerHost(config.getMaxConcurrentRequestsPerHost());
        httpClientBuilder.dispatcher(dispatcher);
      }

      // Always use NO_PROXY
      httpClientBuilder.proxy(Proxy.NO_PROXY);

      if (config.getUserAgent() != null && !config.getUserAgent().isEmpty()) {
        httpClientBuilder.addNetworkInterceptor(chain -> {
          Request agent = chain.request().newBuilder().header("User-Agent", config.getUserAgent()).build();
          return chain.proceed(agent);
        });
      }

      if (config.getTlsVersions() != null && config.getTlsVersions().length > 0) {
        ConnectionSpec spec = new ConnectionSpec.Builder(ConnectionSpec.MODERN_TLS)
            .tlsVersions(config.getTlsVersions())
            .build();
        httpClientBuilder.connectionSpecs(Arrays.asList(spec, CLEARTEXT));
      }

      return httpClientBuilder.build();
    } catch (Exception e) {
      LOGGER.error("Failed to initialize the Kubernetes client: " + e.getMessage(), e);
      fatalErrorHandler.systemExit(-1);
      return null; // will not happen, because we called System.exit()
    }
  }
}
