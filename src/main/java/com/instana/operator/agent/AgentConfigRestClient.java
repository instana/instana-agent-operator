package com.instana.operator.agent;

import io.reactivex.Maybe;
import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.LockSupport;

@ApplicationScoped
public class AgentConfigRestClient {

  private static final Logger LOGGER = LoggerFactory.getLogger(AgentConfigRestClient.class);
  private static final MediaType YAML = MediaType.get("text/yaml");

  private static final int RETRY_COUNT = 6;
  private static final int RETRY_DELAY_SECONDS = 30;

  private OkHttpClient client = new OkHttpClient();

  public Maybe<Void> updateAgentLeaderStatus(String agentIp, int agentPort, boolean isLeader) {
    Throwable error = null;
    for (int i = 0; i < RETRY_COUNT; i++) {
      LockSupport.parkNanos(TimeUnit.SECONDS.toNanos((RETRY_DELAY_SECONDS)));
      LOGGER.debug("Try {} updating agent config...", i + 1);
      RequestBody body = RequestBody.create(YAML, "com.instana.plugin.kubernetes.leader: " + isLeader);
      Request request = new Request.Builder()
          .url("http://" + agentIp + ":" + agentPort + "/config/com.instana.plugin.kubernetes")
          .post(body)
          .build();
      try {
        client.newCall(request).execute().close();
        return Maybe.empty();
      } catch (Throwable t) {
        error = t;
      }
    }

    return Maybe.error(error); // error cannot be null here.
  }
}
