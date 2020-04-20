local operatorResources = std.parseJson(std.extVar('operatorResources'));
local version = std.extVar('version');

local addVersionToMetadataLabels(resource) = resource + {
	metadata+: { labels+:
	 super.labels + { "app.kubernetes.io/version": version }
	}
};
local operatorResourcesWithVersion = std.map(addVersionToMetadataLabels, operatorResources);

local addVersionToDeploymentSpec(deployment) = deployment + {
  spec+: {
    selector+: { matchLabels+:
      super.matchLabels + { "app.kubernetes.io/version": version }
  	},
  	template+: { metadata+: { labels+:
  	  super.labels + { "app.kubernetes.io/version": version }
  	}}
  }
};
local isDeployment(res) = res.kind == "Deployment";
local notDeployment(res) = res.kind != "Deployment";
local deploymentWithVersion = std.filterMap(isDeployment, addVersionToDeploymentSpec, operatorResourcesWithVersion);
local operatorResourcesWithoutDeployment = std.filter(notDeployment, operatorResourcesWithVersion);

{
  ["namespace.v" + version + ".json"]: std.filter(function(res) res.kind == "Namespace", operatorResourcesWithoutDeployment)[0],
  ["serviceaccount.v" + version + ".json"]: std.filter(function(res) res.kind == "ServiceAccount", operatorResourcesWithoutDeployment)[0],
  ["clusterrole.v" + version + ".json"]: std.filter(function(res) res.kind == "ClusterRole", operatorResourcesWithoutDeployment)[0],
  ["clusterrolebinding.v" + version + ".json"]: std.filter(function(res) res.kind == "ClusterRoleBinding", operatorResourcesWithoutDeployment)[0],
  ["customresourcedefinition.v" + version + ".json"]: std.filter(function(res) res.kind == "CustomResourceDefinition", operatorResourcesWithoutDeployment)[0],
  ["deployment.v" + version + ".json"]: deploymentWithVersion[0]
}
