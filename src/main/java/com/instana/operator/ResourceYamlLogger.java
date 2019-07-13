package com.instana.operator;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import io.fabric8.kubernetes.api.model.HasMetadata;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.enterprise.context.ApplicationScoped;
import javax.inject.Inject;

@ApplicationScoped
public class ResourceYamlLogger {

  private static final Logger LOGGER = LoggerFactory.getLogger(ResourceYamlLogger.class);

  @Inject
  FatalErrorHandler fatalErrorHandler;

  void log(HasMetadata resource) {
    if (LOGGER.isTraceEnabled()) {
      try {
        ObjectMapper mapper = new ObjectMapper(new YAMLFactory());
        String yaml = mapper.writerWithDefaultPrettyPrinter().writeValueAsString(resource);
        LOGGER.trace("Creating resource:\n" + yaml);
      } catch (Exception e) {
        LOGGER.error("Failed to serialize resource to YAML: " + e.getMessage(), e);
        fatalErrorHandler.systemExit(-1);
      }
    }
  }
}
