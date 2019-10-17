local crd = std.parseJson(std.extVar('crds'))[0];
local crd_descriptors = std.parseJson(std.extVar('crd_descriptors'))[0];
local description = std.extVar('description');
local examples = std.extVar('examples');
local image = std.extVar('image');
local isoDate = std.extVar('isoDate');
local resources = std.parseJson(std.extVar('resources'));
local version = std.extVar('version');
local prevVersion = std.extVar('prevVersion');
local crdVersion = "v1beta1";

local isClusterRole(res) = res.kind == "ClusterRole";
local rules = std.filter(isClusterRole, resources)[0].rules;
local isDeployment(res) = res.kind == "Deployment";
local mapDeployment(dep) = {
	name: dep.metadata.name,
	spec: dep.spec
};
local deployment = std.filterMap(isDeployment, mapDeployment, resources)[0];
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
	["instana-agent.v" + version + ".clusterserviceversion.json"]: {
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind": "ClusterServiceVersion",
		"metadata": {
			"annotations": {
				"alm-examples": examples,
				"categories": "Monitoring,OpenShift Optional",
				"certified": "false",
				"containerImage": "instana/instana-agent-operator:" + version,
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
			"replaces": "instana-agent-operator.v" + prevVersion,
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
	"instana-agent.package.json": {
		packageName: "instana-agent",
		channels: [
			{
				name: "beta",
				currentCSV: "instana-agent-operator.v" + version
			}
		]
	},
	["instana-agent.crd.json"]: crd
}
