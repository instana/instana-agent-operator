/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc.
 */
package com.instana.operator.customresource;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import io.fabric8.kubernetes.client.CustomResourceList;
import io.quarkus.runtime.annotations.RegisterForReflection;

@JsonDeserialize
@JsonIgnoreProperties(ignoreUnknown = true)
@RegisterForReflection
public class InstanaAgentList extends CustomResourceList<InstanaAgent> {}
