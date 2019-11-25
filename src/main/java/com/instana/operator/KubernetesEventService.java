package com.instana.operator;

import io.fabric8.kubernetes.api.model.Event;
import io.fabric8.kubernetes.api.model.HasMetadata;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;
import java.io.InputStream;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;

import static com.instana.operator.resource.KubernetesResource.name;

@ApplicationScoped
public class KubernetesEventService {

  private static final Logger LOGGER = LoggerFactory.getLogger(KubernetesEventService.class);
  private final DateTimeFormatter UTC = DateTimeFormatter.ISO_ZONED_DATE_TIME;

  @Inject
  DefaultKubernetesClient client;
  @Inject
  ResourceYamlLogger yamlLogger;

  public void createKubernetesEvent(String namespace, String reason, String message, HasMetadata involvedObject) {
    try {
      String now = nowUTC();
      InputStream in = getClass().getResourceAsStream("/event.yaml");
      Event event = client.inNamespace(namespace).events().load(in).get();
      event.getMetadata().setNamespace(namespace);
      event.setFirstTimestamp(now);
      event.setLastTimestamp(now);
      event.setReason(reason);
      event.setMessage(message);
      event.getInvolvedObject().setApiVersion(involvedObject.getApiVersion());
      event.getInvolvedObject().setKind(involvedObject.getKind());
      event.getInvolvedObject().setNamespace(involvedObject.getMetadata().getNamespace());
      event.getInvolvedObject().setName(involvedObject.getMetadata().getName());
      event.getInvolvedObject().setUid(involvedObject.getMetadata().getUid());
      Event eventWithName = client.inNamespace(namespace).events().create(event);
      yamlLogger.log(event);
      LOGGER.debug("Created Kubernetes event " + name(eventWithName));
    } catch (Exception e) {
      LOGGER.warn("Failed to create Kubernetes event in namespace " + namespace + ": " + e.getMessage(), e);
      // This is not worth a System.exit(). Ignore it and carry on.
    }
  }

  private String nowUTC() {
    ZonedDateTime now = ZonedDateTime.now(ZoneOffset.UTC);
    return UTC.format(now);
  }
}
