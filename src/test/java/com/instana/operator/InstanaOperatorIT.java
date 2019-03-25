package com.instana.operator;

import static io.restassured.RestAssured.given;
import static org.hamcrest.Matchers.is;

import javax.inject.Inject;

import org.eclipse.microprofile.config.inject.ConfigProperty;
import org.junit.jupiter.api.Test;

import com.instana.operator.resource.VersionResource;

import io.quarkus.test.junit.QuarkusTest;

@QuarkusTest
class InstanaOperatorIT {

  @Inject
  VersionResource versionResource;

  @Test
  void healthResourceShouldReturnUP() {
    given()
        .when().get("/health")
        .then()
        .statusCode(200)
        .body("checks[0].state", is("UP"));
  }

  @Test
  void versionResourceShouldReturnVersion() {
    given()
        .when().get("/version")
        .then()
        .statusCode(200)
        .body(is(versionResource.getVersion()));
  }

}