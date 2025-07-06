/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc. 2024
*/

package backend

func NewK8SensorBackend(
	ResourceSuffix string,
	EndpointKey string,
	DownloadKey string,
	EndpointHost string,
	EndpointPort string,
) *K8SensorBackend {
	return &K8SensorBackend{
		ResourceSuffix: ResourceSuffix,
		EndpointKey:    EndpointKey,
		DownloadKey:    DownloadKey,
		EndpointHost:   EndpointHost,
		EndpointPort:   EndpointPort,
	}
}

type K8SensorBackend struct {
	ResourceSuffix string
	EndpointKey    string
	DownloadKey    string
	EndpointHost   string
	EndpointPort   string
}

func NewRemoteSensorBackend(
	ResourceSuffix string,
	EndpointKey string,
	DownloadKey string,
	EndpointHost string,
	EndpointPort string,
) *RemoteSensorBackend {
	return &RemoteSensorBackend{
		ResourceSuffix: ResourceSuffix,
		EndpointKey:    EndpointKey,
		DownloadKey:    DownloadKey,
		EndpointHost:   EndpointHost,
		EndpointPort:   EndpointPort,
	}
}

type RemoteSensorBackend struct {
	ResourceSuffix string
	EndpointKey    string
	DownloadKey    string
	EndpointHost   string
	EndpointPort   string
}
