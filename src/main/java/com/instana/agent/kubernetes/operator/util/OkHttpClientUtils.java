package com.instana.agent.kubernetes.operator.util;

import static okhttp3.ConnectionSpec.CLEARTEXT;
import static org.apache.commons.lang3.StringUtils.isEmpty;

import java.net.Proxy;
import java.security.GeneralSecurityException;
import java.util.Arrays;
import java.util.concurrent.TimeUnit;

import javax.net.ssl.KeyManager;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;

import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.fabric8.kubernetes.client.Config;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.internal.SSLUtils;
import io.fabric8.kubernetes.client.utils.BackwardsCompatibilityInterceptor;
import io.fabric8.kubernetes.client.utils.ImpersonatorInterceptor;
import io.fabric8.kubernetes.client.utils.Utils;
import okhttp3.ConnectionSpec;
import okhttp3.Credentials;
import okhttp3.Dispatcher;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.logging.HttpLoggingInterceptor;

public abstract class OkHttpClientUtils {

  private OkHttpClientUtils() {
  }

  public static OkHttpClient createHttpClient(final Config config) {
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

        try {
          SSLContext sslContext = SSLUtils.sslContext(keyManagers, trustManagers, config.isTrustCerts());
          httpClientBuilder.sslSocketFactory(sslContext.getSocketFactory(), trustManager);
        } catch (GeneralSecurityException e) {
          throw new AssertionError(); // The system has no TLS. Just give up.
        }
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
      throw KubernetesClientException.launderThrowable(e);
    }
  }

}
