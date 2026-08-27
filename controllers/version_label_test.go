/*
(c) Copyright IBM Corp. 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOperatorVersionLabelInDeployment verifies that the operator-version label
// is present in the deployment metadata and pod template, but NOT in the selector
func TestOperatorVersionLabelInDeployment(t *testing.T) {
	// Generate the controller YAML with a test version
	testVersion := "3.0.0-test"
	cmd := exec.Command("make", "controller-yaml", "VERSION="+testVersion)
	cmd.Dir = ".."
	output, err := cmd.Output()
	require.NoError(t, err, "Failed to generate controller YAML")

	yamlContent := string(output)

	// Test 1: Verify no placeholders remain
	assert.NotContains(t, yamlContent, "OPERATOR_VERSION_PLACEHOLDER",
		"Placeholder should be replaced in generated YAML")

	// Test 2: Verify operator-version label exists with correct value
	expectedLabel := "app.kubernetes.io/operator-version: " + testVersion
	assert.Contains(t, yamlContent, expectedLabel,
		"operator-version label with correct version should exist")

	// Test 3: Count occurrences (should be 2: deployment metadata + pod template)
	count := strings.Count(yamlContent, expectedLabel)
	assert.Equal(t, 2, count,
		"operator-version label should appear exactly twice (deployment metadata + pod template)")

	// Test 4: Verify label NOT in selector section
	// Extract the selector section and verify it doesn't contain operator-version
	selectorRegex := regexp.MustCompile(`(?s)selector:\s*\n\s*matchLabels:.*?(?:\n\s{2}\S|\nspec:)`)
	selectorMatches := selectorRegex.FindAllString(yamlContent, -1)
	for _, selectorSection := range selectorMatches {
		assert.NotContains(t, selectorSection, "operator-version",
			"operator-version should NOT be in selector (immutable field)")
	}
}

// TestOperatorVersionLabelInBundle verifies that the operator-version label
// is present in the OLM bundle manifests
func TestOperatorVersionLabelInBundle(t *testing.T) {
	// Check if bundle directory exists
	bundleDir := "../bundle/manifests"
	if _, err := os.Stat(bundleDir); os.IsNotExist(err) {
		t.Skip("Bundle not generated, run 'make bundle' first")
	}

	// Read the CSV file
	csvPath := bundleDir + "/instana-agent-operator.clusterserviceversion.yaml"
	csvData, err := os.ReadFile(csvPath)
	require.NoError(t, err, "Failed to read CSV file")

	csvContent := string(csvData)

	// Test 1: Verify no placeholders remain
	assert.NotContains(t, csvContent, "OPERATOR_VERSION_PLACEHOLDER",
		"Placeholder should be replaced in bundle")

	// Test 2: Verify operator-version label exists
	assert.Contains(t, csvContent, "app.kubernetes.io/operator-version",
		"operator-version label should exist in CSV")

	// Test 3: Verify the label has a value (not empty)
	lines := strings.Split(csvContent, "\n")
	foundLabel := false
	for _, line := range lines {
		if strings.Contains(line, "app.kubernetes.io/operator-version:") {
			foundLabel = true
			// Extract the value part after the colon
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				assert.NotEmpty(t, value, "operator-version value should not be empty")
				assert.NotEqual(t, "OPERATOR_VERSION_PLACEHOLDER", value,
					"placeholder should be replaced")
			}
			break
		}
	}
	assert.True(t, foundLabel, "operator-version label should be found in CSV")
}

// Made with Bob
