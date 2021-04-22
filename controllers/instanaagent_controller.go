/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package controllers

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"

	"io/ioutil"

	"github.com/go-logr/logr"
	instanaV1Beta1 "github.com/instana/instana-agent-operator/api/v1beta1"
	"github.com/pkg/errors"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AppName                  = "instana-agent"
	AppVersion               = "1.0.0-beta"
	AgentKey                 = "key"
	AgentDownloadKey         = "downloadKey"
	DefaultAgentImageName    = "instana/agent"
	AgentImagePullSecretName = "containers-key"
	DockerRegistry           = "containers.instana.io"

	AgentNameSpace          = AppName
	AgentSecretName         = AppName
	AgentServiceAccountName = AppName
	AgentPort               = 42699
	OpenTelemetryPort       = 55680
)

// InstanaAgentReconciler reconciles a InstanaAgent object
type InstanaAgentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=agents.instana.com,namespace=instana-agent,resources=instanaagent/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,namespace=instana-agent,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,namespace=instana-agent,resources=pods,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InstanaAgent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("instanaagent", req.NamespacedName)

	// Fetch the InstanaAgent CRD instance
	crdInstance := &instanaV1Beta1.InstanaAgent{}
	err := r.Get(ctx, req.NamespacedName, crdInstance)
	if err != nil {
		if k8sError.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	r.Log.Info("Instana CRD fetched successfully")
	//create secrets
	// createSecrets(r.Client, crdInstance, r.Log)

	//create service account
	r.Log.Info("creating service account")
	createService(r.Client, crdInstance, r.Log)
	createServiceAccount(r.Client, crdInstance, r.Log)

	createConfigMap(r.Client, crdInstance, r.Log)
	createClusterRole(r.Client, crdInstance, r.Log)
	createClusterRoleBinding(r.Client, crdInstance, r.Log)

	//check if the daemonset already deployed, if not create a new one.
	foundDaemonset := &appV1.DaemonSet{}
	err = r.Get(ctx, req.NamespacedName, foundDaemonset)
	if err != nil {
		if k8sError.IsNotFound(err) {
			r.Log.Info("No Daemonset deployed before, creating new one")
			newDaemonset := newDaemonsetForCRD(crdInstance)
			if err := r.Create(ctx, newDaemonset); err != nil {
				r.Log.Error(err, "Failed to create Daemonset")
			} else {
				r.Log.Info("Daemonset created successfully")
			}
		}
	}

	return ctrl.Result{}, nil
}

//Todo: refactor all resource object creation chains into another component
func createClusterRole(c client.Client, crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) {
	clusterRole := rbacV1.ClusterRole{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   AppName,
			Labels: buildLabels(),
		},
		Rules: []rbacV1.PolicyRule{
			{
				NonResourceURLs: []string{"/version"},
				Verbs:           []string{"get"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs", "cronjobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"extensions"},
				Resources: []string{"deployments", "replicasets", "ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "replicasets", "daemonsets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "events", "services", "endpoints", "nodes", "pods", "replicationcontrollers",
					"componentstatuses", "resourcequotas", "persistentvolumes", "persistentvolumeclaims"},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	if err := c.Create(context.TODO(), &clusterRole); err != nil {
		Log.Error(err, "Failed to create clusterRole")
	} else {
		Log.Info("ClusterRole created successfully")
	}
}

func createClusterRoleBinding(c client.Client, crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) {
	clusterRoleBinding := rbacV1.ClusterRoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   AppName,
			Labels: buildLabels(),
		},
		Subjects: []rbacV1.Subject{{
			Kind:      "ServiceAccount",
			Name:      AppName,
			Namespace: AgentNameSpace,
		}},
		RoleRef: rbacV1.RoleRef{
			Kind:     "ClusterRole",
			Name:     AppName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	if err := c.Create(context.TODO(), &clusterRoleBinding); err != nil {
		Log.Error(err, "Failed to create clusterRoleBinding")
	} else {
		Log.Info("ClusterRoleBinding created successfully")
	}
}

func createServiceAccount(c client.Client, crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) {
	serviceAccount := coreV1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		}}
	if err := c.Create(context.TODO(), &serviceAccount); err != nil {
		Log.Error(err, "Failed to create Service account")
	} else {
		Log.Info("Service account created successfully")
	}
}

func createService(c client.Client, crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) {
	service := coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Spec: coreV1.ServiceSpec{
			Selector: buildLabels(),
			Ports: []coreV1.ServicePort{
				{
					Name:       "opentelemetry",
					Protocol:   coreV1.ProtocolTCP,
					Port:       OpenTelemetryPort,
					TargetPort: intstr.FromInt(OpenTelemetryPort),
				},
				{
					Name:       "agent-apis",
					Protocol:   coreV1.ProtocolTCP,
					Port:       AgentPort,
					TargetPort: intstr.FromInt(AgentPort),
				},
			},
			TopologyKeys: []string{"kubernetes.io/hostname"},
		},
	}
	if err := c.Create(context.TODO(), &service); err != nil {
		Log.Error(err, "Failed to create Service")
	} else {
		Log.Info("Service created successfully")
	}
}

// returns a Daemonset object with the data hold in instanaAgent crd instance
func newDaemonsetForCRD(crdInstance *instanaV1Beta1.InstanaAgent) *appV1.DaemonSet {
	//we need to have a same matched label for all our agent resources
	selectorLabels := buildLabels()
	podSpec := newPodSpec(crdInstance)
	return &appV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      crdInstance.Name,
			Namespace: crdInstance.Namespace,
			Labels:    selectorLabels,
		},
		Spec: appV1.DaemonSetSpec{
			Selector: &metaV1.LabelSelector{MatchLabels: selectorLabels},
			UpdateStrategy: appV1.DaemonSetUpdateStrategy{
				Type:          appV1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appV1.RollingUpdateDaemonSet{MaxUnavailable: &intstr.IntOrString{IntVal: 1}},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: podSpec,
			},
		},
	}
}

func createSecrets(c client.Client, crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) {
	Log.Info("Creating InstanaAgent config secret")
	if err := c.Create(context.TODO(), &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AgentSecretName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Type: coreV1.SecretTypeOpaque,
		Data: map[string][]byte{
			AgentKey:         []byte(crdInstance.Spec.Key),
			AgentDownloadKey: []byte(crdInstance.Spec.DownloadKey),
		},
	}); err != nil {
		Log.Error(err, "failed to create secret %s", AgentSecretName)
	}

	Log.Info("Creating ImagePullSecret secret")
	err := c.Create(context.TODO(), &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AgentImagePullSecretName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Type: coreV1.SecretTypeDockerConfigJson,
		Data: generatePullSecretData(c, crdInstance, Log),
	})
	if err != nil {
		Log.Error(err, "failed to create secret %s", AgentSecretName)
	}
}
func generatePullSecretData(c client.Client, crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) map[string][]byte {
	type auths struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Auth     string `json:"auth,omitempty"`
	}

	type dockerConfig struct {
		Auths map[string]auths `json:"auths,omitempty"`
	}
	passwordKey := crdInstance.Spec.Key
	if len(passwordKey) == 0 {
		passwordKey = crdInstance.Spec.DownloadKey
	}
	a := fmt.Sprintf("%s:%s", "_", passwordKey)
	a = b64.StdEncoding.EncodeToString([]byte(a))

	auth := auths{
		Username: "_",
		Password: passwordKey,
		Auth:     a,
	}

	d := dockerConfig{
		Auths: map[string]auths{
			DockerRegistry: auth,
		},
	}
	j, err := json.Marshal(d)
	if err != nil {
		Log.Error(errors.WithStack(err), "Failed to convert jsonkey")
	}

	return map[string][]byte{".dockerconfigjson": j}
}

func createConfigMap(c client.Client, crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) {
	Log.Info("Creating InstanaAgent config map")
	if err := c.Create(context.TODO(), &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Data: map[string]string{
			"cluster_name":       crdInstance.Spec.ClusterName,
			"configuration.yaml": readFile("configuration.yaml", Log),
		},
	}); err != nil {
		Log.Error(err, "failed to create configMap %s", AppName)
	} else {
		Log.Info("Service created successfully")
	}
}

func readFile(filename string, Log logr.Logger) string {
	content, err := ioutil.ReadFile("config/" + filename)
	if err != nil {
		Log.Error(err, "failed to read configuration.yaml")
	}
	return string(content)
}

func newPodSpec(crdInstance *instanaV1Beta1.InstanaAgent) coreV1.PodSpec {

	trueVar := true
	secCtx := &coreV1.SecurityContext{
		Privileged: &trueVar,
	}

	AgentImageName := DefaultAgentImageName
	if len(crdInstance.Spec.Image) > 0 {
		AgentImageName = crdInstance.Spec.Image
	}

	return coreV1.PodSpec{
		ServiceAccountName: AgentServiceAccountName,
		HostIPC:            true,
		HostNetwork:        true,
		HostPID:            true,
		DNSPolicy:          coreV1.DNSClusterFirstWithHostNet,
		ImagePullSecrets:   []coreV1.LocalObjectReference{{Name: AgentImagePullSecretName}},
		Containers: []coreV1.Container{{
			Name:            AppName,
			Image:           AgentImageName,
			ImagePullPolicy: coreV1.PullAlways,
			Env:             buildEnvVars(crdInstance),
			SecurityContext: secCtx,
			Ports:           []coreV1.ContainerPort{{ContainerPort: AgentPort}},
			VolumeMounts:    buildVolumeMounts(crdInstance),
			LivenessProbe: &coreV1.Probe{
				InitialDelaySeconds: 300,
				TimeoutSeconds:      3,
				Handler: coreV1.Handler{
					HTTPGet: &coreV1.HTTPGetAction{
						Path: "/status",
						Port: intstr.FromInt(AgentPort),
					}}},
		}},
		Volumes:     buildVolumes(crdInstance),
		Tolerations: []coreV1.Toleration{},
	}
}

func buildLabels() map[string]string {
	return map[string]string{
		"app":                          AppName,
		"app.kubernetes.io/name":       AppName,
		"app.kubernetes.io/version":    AppVersion,
		"app.kubernetes.io/managed-by": AppName,
	}
}

func buildEnvVars(crdInstance *instanaV1Beta1.InstanaAgent) []coreV1.EnvVar {
	envVars := crdInstance.Spec.Env

	// optional := true
	agentEnvVars := []coreV1.EnvVar{
		{Name: "INSTANA_OPERATOR_MANAGED", Value: "true"},
		// {Name: "INSTANA_AGENT_LEADER_ELECTOR_PORT", Value: "42699"},
		{Name: "INSTANA_ZONE", Value: crdInstance.Spec.ZoneName},
		{Name: "INSTANA_KUBERNETES_CLUSTER_NAME", Value: crdInstance.Spec.ClusterName},
		{Name: "INSTANA_AGENT_ENDPOINT", Value: crdInstance.Spec.Endpoint.Host},
		{Name: "INSTANA_AGENT_ENDPOINT_PORT", Value: crdInstance.Spec.Endpoint.Port},
		{Name: "INSTANA_AGENT_KEY", ValueFrom: &coreV1.EnvVarSource{
			SecretKeyRef: &coreV1.SecretKeySelector{
				LocalObjectReference: coreV1.LocalObjectReference{Name: AgentSecretName},
				Key:                  "key",
			},
		}},
		// {Name: "INSTANA_DOWNLOAD_KEY", ValueFrom: &coreV1.EnvVarSource{
		// 	SecretKeyRef: &coreV1.SecretKeySelector{
		// 		LocalObjectReference: coreV1.LocalObjectReference{Name: AgentSecretName},
		// 		Key:                  "downloadKey",
		// 		Optional:             &optional,
		// 	},
		// }},
		{Name: "INSTANA_AGENT_POD_NAME", ValueFrom: &coreV1.EnvVarSource{
			FieldRef: &coreV1.ObjectFieldSelector{
				FieldPath:  "metadata.name",
				APIVersion: "v1",
			},
		}},
		{Name: "POD_IP", ValueFrom: &coreV1.EnvVarSource{
			FieldRef: &coreV1.ObjectFieldSelector{
				FieldPath:  "status.podIP",
				APIVersion: "v1",
			},
		}},
	}
	return append(envVars, agentEnvVars...)
}

func buildVolumeMounts(instance *instanaV1Beta1.InstanaAgent) []coreV1.VolumeMount {
	return []coreV1.VolumeMount{
		{
			Name:      "dev",
			MountPath: "/dev",
		},
		{
			Name:      "run",
			MountPath: "/run",
		},
		{
			Name:      "var-run",
			MountPath: "/var/run",
		},
		{
			Name:      "var-run-kubo",
			MountPath: "/var/vcap/sys/run/docker",
		},
		{
			Name:      "sys",
			MountPath: "/sys",
		},
		{
			Name:      "var-log",
			MountPath: "/var/log",
		},
		{
			Name:      "var-lib",
			MountPath: "/var/lib/containers/storage",
		},
		{
			Name:      "machine-id",
			MountPath: "/etc/machine-id",
		},
		{
			Name:      "configuration",
			SubPath:   "configuration.yaml",
			MountPath: "/root/configuration.yaml",
		},
	}
}

func buildVolumes(instance *instanaV1Beta1.InstanaAgent) []coreV1.Volume {
	return []coreV1.Volume{
		{
			Name: "dev",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/",
				},
			},
		},
		{
			Name: "run",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/run",
				},
			},
		},
		{
			Name: "var-run",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/run",
				},
			},
		},
		{
			Name: "var-run-kubo",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/vcap/sys/run/docker",
				},
			},
		},
		{
			Name: "sys",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/sys",
				},
			},
		},
		{
			Name: "var-log",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "var-lib",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/var/lib/containers/storage",
				},
			},
		},
		{
			Name: "machine-id",
			VolumeSource: coreV1.VolumeSource{
				HostPath: &coreV1.HostPathVolumeSource{
					Path: "/etc/machine-id",
				},
			},
		},
		{
			Name: "configuration",
			VolumeSource: coreV1.VolumeSource{
				ConfigMap: &coreV1.ConfigMapVolumeSource{LocalObjectReference: coreV1.LocalObjectReference{Name: AppName}},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstanaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instanaV1Beta1.InstanaAgent{}).
		Owns(&appV1.DaemonSet{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.Pod{}).
		Complete(r)
}
