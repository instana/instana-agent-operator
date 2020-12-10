local crd = std.parseJson(std.extVar('crds'))[0];
local crd_descriptors = std.parseJson(std.extVar('crd_descriptors'))[0];
local description = std.extVar('description');
local examples = std.extVar('examples');
local image = std.extVar('image');
local isoDate = std.extVar('isoDate');
local redhat = std.parseJson(std.extVar('redhat'));
local registry = std.extVar('registry');
local replaces = std.extVar('replaces');
local resources = std.parseJson(std.extVar('resources'));
local version = std.extVar('version');

local crdVersion = "v1beta1";
local imagePrefix = if std.length(registry) > 0 then registry + "/" else "";
local renderCsvVersion = !redhat;
local isClusterRole(res) = res.kind == "ClusterRole";
local rules = std.filter(isClusterRole, resources)[0].rules;
local isDeployment(res) = res.kind == "Deployment";
local mapDeployment(dep) = {
	name: dep.metadata.name,
	spec: dep.spec
};
local deployment = std.filterMap(isDeployment, mapDeployment, resources)[0] + {
	spec+: { template+: { spec+: {
		assert std.length(super.containers) == 1 : "Expected exactly 1 container in operator deployment pod",
		containers: [
			super.containers[0] {
				image: imagePrefix + super.image,
				[if redhat then "ports"]: [{ containerPort: 9000 }],
				[if redhat then "env"]: super.env + [
					{
						name: "RELATED_IMAGE_INSTANA_AGENT",
						value: "registry.connect.redhat.com/instana/agent:latest"
					},
					{
						name: "QUARKUS_HTTP_PORT",
						value: "9000"
					}
				]
			}
		]
	}}}
};
local isCrd(res) = res.kind == "CustomResourceDefinition";
local crd = std.filter(isCrd, resources)[0];
local crdWithDescriptors = {
	description: "Instana Agent",
	displayName: "Instana Agent",
	name: crd.metadata.name,
	kind: crd.spec.names.kind,
	specDescriptors: crd_descriptors.specDescriptors,
	version: crdVersion
};

{
	["instana-agent-operator" + (if renderCsvVersion then ".v" + version else "") + ".clusterserviceversion.json"]: {
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind": "ClusterServiceVersion",
		"metadata": {
			"annotations": {
				"alm-examples": examples,
				"categories": "Monitoring,OpenShift Optional",
				"certified": "false",
				"containerImage": imagePrefix + "instana/instana-agent-operator:" + version,
				"createdAt": isoDate,
				"description": "Fully automated Application Performance Monitoring (APM) for microservices.",
				"support": "Instana",
				"repository": "https://github.com/instana/instana-agent-operator",
				"capabilities": "Basic Install"
			},
			"name": "instana-agent-operator.v" + version,
			"namespace": "placeholder"
		},
		"spec": {
			"displayName": "Instana Agent Operator",
			"description": description,
			"icon": [
				{
					"base64data": std.base64(image),
					"mediatype": "image/svg+xml"
				}
			],
			"version": version,
			"replaces": replaces,
			"minKubeVersion": "1.11.0",
			"provider": {
				"name": "Instana"
			},
			"links": [
				{
					"name": "GitHub Repository",
					"url": "https://github.com/instana/instana-agent-operator"
				}
			],
			"keywords": [
				"monitoring",
				"APM",
				"Instana"
			],
			"maintainers": [
				{
					"email": "support@instana.com",
					"name": "Instana"
				}
			],
			"maturity": "beta",
			"apiservicedefinitions": {},
			"customresourcedefinitions": {
				"owned": [
					crdWithDescriptors
				]
			},
			"install": {
				"strategy": "deployment",
				"spec": {
					"clusterPermissions": [
						{
							"serviceAccountName": "instana-agent-operator",
							"rules": rules
						}
					],
					"deployments": [deployment]
				}
			},
			"installModes": [
				{
					"type": "OwnNamespace",
					"supported": true
				},
				{
					"type": "SingleNamespace",
					"supported": true
				},
				{
					"type": "MultiNamespace",
					"supported": true
				},
				{
					"type": "AllNamespaces",
					"supported": true
				}
			]
		}
	},
	[if !redhat then "instana-agent.package.json"]: {
		packageName: "instana-agent",
		channels: [
			{
				name: "beta",
				currentCSV: "instana-agent-operator.v" + version
			}
		]
	},
	"agents.instana.io.crd.json": crd
}
