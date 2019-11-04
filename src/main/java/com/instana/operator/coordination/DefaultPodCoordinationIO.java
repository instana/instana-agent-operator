package com.instana.operator.coordination;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.fabric8.kubernetes.api.model.Pod;
import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
import javax.inject.Named;
import java.io.IOException;
import java.util.Set;

import static com.instana.operator.client.KubernetesClientProducer.AGENT_POD_HTTP_CLIENT;

@ApplicationScoped
class DefaultPodCoordinationIO implements PodCoordinationIO {
  private static final ObjectMapper MAPPER = new ObjectMapper();
  private static final int AGENT_PORT = 42699;

  @Inject
  @Named(AGENT_POD_HTTP_CLIENT)
  OkHttpClient httpClient;

  @Override
  public void assign(Pod pod, Set<String> assignment) throws IOException {
    MediaType mediaType = MediaType.get("application/json");
    Request req = new Request.Builder()
        .url(baseUrl(pod) + "/assigned")
        .put(RequestBody.create(mediaType, MAPPER.writeValueAsBytes(assignment)))
        .build();

    try (Response res = httpClient.newCall(req).execute()) {
      if (!res.isSuccessful()) {
        throw new IOException("Unsuccessful assignment of resource leadership to "
            + pod.getMetadata().getName() + ": " + res.code() + " " + res.message());
      }
    }
  }

  @Override
  public CoordinationRecord pollPod(Pod pod) throws IOException {
    Request req = new Request.Builder()
        .url(baseUrl(pod))
        .get()
        .build();

    try (Response res = httpClient.newCall(req).execute()) {
      if (!res.isSuccessful()) {
        throw new IOException("Unsuccessful request polling "
            + pod.getMetadata().getName() + ": " + res.code() + " " + res.message());

      }

      if (res.body() == null) {
        throw new IOException("Unexpected empty body polling " + pod.getMetadata().getName());
      }

      return MAPPER.readValue(res.body().bytes(), CoordinationRecord.class);
    }
  }

  String baseUrl(Pod pod) {
    String ip = pod.getStatus().getHostIP();

    return "http://" + ip + ":" + AGENT_PORT + "/coordination";
  }
}
