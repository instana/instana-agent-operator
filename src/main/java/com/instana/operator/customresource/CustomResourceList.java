package com.instana.operator.customresource;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.api.model.KubernetesResource;
import io.fabric8.kubernetes.api.model.KubernetesResourceList;
import io.fabric8.kubernetes.api.model.ListMeta;

import javax.validation.Valid;
import javax.validation.constraints.NotNull;
import java.util.ArrayList;
import java.util.List;

// This class is copied from https://github.com/fabric8io/kubernetes-client/pull/1616
// As soon as that PR is merged, this class can be removed and replaced with
// io.fabric8.kubernetes.client.CustomResourceList.

/**
 */
@JsonDeserialize(using = com.fasterxml.jackson.databind.JsonDeserializer.None.class)
public class CustomResourceList<T extends HasMetadata> implements KubernetesResource, KubernetesResourceList<T> {

  @NotNull
  @JsonProperty("apiVersion")
  private String apiVersion;
  @JsonProperty("items")
  @Valid
  private List<T> items = new ArrayList<T>();
  @NotNull
  @JsonProperty("kind")
  private String kind;
  @JsonProperty("metadata")
  @Valid
  private ListMeta metadata;

  public String getApiVersion() {
    return apiVersion;
  }

  public void setApiVersion(String apiVersion) {
    this.apiVersion = apiVersion;
  }

  @Override
  public List<T> getItems() {
    return items;
  }

  public void setItems(List<T> items) {
    this.items = items;
  }

  public String getKind() {
    return kind;
  }

  public void setKind(String kind) {
    this.kind = kind;
  }

  @Override
  public ListMeta getMetadata() {
    return metadata;
  }

  public void setMetadata(ListMeta metadata) {
    this.metadata = metadata;
  }
}
