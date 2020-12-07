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
    template+: {
      metadata+: {
        labels+: super.labels + { "app.kubernetes.io/version": version }
      },
      spec+:
        super.spec + {
          containers: std.mapWithIndex(function(i, c) if i == 0 then c + {image: c.image + ":" + version} else c, super.containers)
        }
    }
  }
};
local isDeployment(res) = res.kind == "Deployment";
local notDeployment(res) = res.kind != "Deployment";
local deploymentWithVersion = std.filterMap(isDeployment, addVersionToDeploymentSpec, operatorResourcesWithVersion);
local operatorResourcesWithoutDeployment = std.filter(notDeployment, operatorResourcesWithVersion);

// These resources need to be this specific order so that it will maintain that order when they are combined back into one file
{
  ["01.namespace.v" + version + ".json"]: std.filter(function(res) res.kind == "Namespace", operatorResourcesWithoutDeployment)[0],
  ["02.serviceaccount.v" + version + ".json"]: std.filter(function(res) res.kind == "ServiceAccount", operatorResourcesWithoutDeployment)[0],
  ["03.clusterrole.v" + version + ".json"]: std.filter(function(res) res.kind == "ClusterRole", operatorResourcesWithoutDeployment)[0],
  ["04.clusterrolebinding.v" + version + ".json"]: std.filter(function(res) res.kind == "ClusterRoleBinding", operatorResourcesWithoutDeployment)[0],
  ["05.customresourcedefinition.v" + version + ".json"]: std.filter(function(res) res.kind == "CustomResourceDefinition", operatorResourcesWithoutDeployment)[0],
  ["06.deployment.v" + version + ".json"]: deploymentWithVersion[0]
}
