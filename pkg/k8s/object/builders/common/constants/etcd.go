/*
(c) Copyright IBM Corp. 2025

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

package constants

import "fmt"

const (
	// ConfigMap and Secret names
	ETCDCASecretName         = "etcd-ca"
	ETCDClientCertSecretName = "etcd-client-cert"

	// ETCD ports
	ETCDMetricsPortHTTPS = 2379
	ETCDMetricsPortHTTP  = 2381
	ETCDOCPMetricsPort   = 9979

	// ETCD environment variables
	EnvETCDTargets        = "ETCD_TARGETS"
	EnvETCDCAFile         = "ETCD_CA_FILE"
	EnvETCDCertFile       = "ETCD_CERT_FILE"
	EnvETCDKeyFile        = "ETCD_KEY_FILE"
	EnvETCDMetricsURL     = "ETCD_METRICS_URL"
	EnvETCDRequestTimeout = "ETCD_REQUEST_TIMEOUT"
	EnvETCDInsecure       = "ETCD_INSECURE"

	// ETCD paths
	ETCDCAMountPath = "/var/run/secrets/etcd"

	// OpenShift ETCD resources
	ETCDMetricsCABundleName    = "etcd-metrics-ca-bundle"
	ETCDMetricClientSecretName = "etcd-metric-client"
	ETCDNamespace              = "openshift-etcd"
	ETCDMetricsCAMountPath     = "/etc/etcd-metrics-ca"
	ETCDClientCertMountPath    = "/etc/etcd-client"

	// Container names
	ContainerK8Sensor = "instana-agent"
)

// ETCD URLs - using functions since constants can't use fmt.Sprintf
func GetETCDOCPMetricsURL() string {
	return fmt.Sprintf(
		"https://etcd.openshift-etcd.svc.cluster.local:%d/metrics",
		ETCDOCPMetricsPort,
	)
}
