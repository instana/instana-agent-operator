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
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AppVersion               = "1.0.0-beta"
	AgentKey                 = "key"
	AgentDownloadKey         = "downloadKey"
	DefaultAgentImageName    = "instana/agent"
	AgentImagePullSecretName = "containers-key"
	DockerRegistry           = "containers.instana.io"

	AgentPort         = 42699
	OpenTelemetryPort = 55680
)

var (
	AppName                 = "instana-agent"
	AgentNameSpace          = AppName
	AgentSecretName         = AppName
	AgentServiceAccountName = AppName
)

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
func (r *InstanaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("instanaagent", req.NamespacedName)

	crdInstance, err := r.fetchCrdInstance(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	} else if crdInstance == nil {
		return ctrl.Result{}, nil
	}

	var reconcilationError = error(nil)

	if err = r.reconcileSecrets(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileImagePullSecrets(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileServices(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileServiceAccounts(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileConfigMap(ctx, crdInstance); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileClusterRole(ctx); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileClusterRoleBinding(ctx); err != nil {
		reconcilationError = err
	}
	if err = r.reconcileDaemonset(ctx, req, crdInstance); err != nil {
		reconcilationError = err
	}

	return ctrl.Result{}, reconcilationError
}

func (r *InstanaAgentReconciler) reconcileDaemonset(ctx context.Context, req ctrl.Request, crdInstance *instanaV1Beta1.InstanaAgent) error {
	daemonset := &appV1.DaemonSet{}
	err := r.Get(ctx, req.NamespacedName, daemonset)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No daemonset deployed before, creating new one")
			daemonset = newDaemonsetForCRD(crdInstance)
			if err := r.Create(ctx, daemonset); err != nil {
				r.Log.Error(err, "Failed to create daemonset")
			} else {
				r.Log.Info(fmt.Sprintf("%s daemonSet created successfully", AppName))
			}
		}
	}
	return err
}

func (r *InstanaAgentReconciler) reconcileClusterRole(ctx context.Context) error {
	clusterRole := &rbacV1.ClusterRole{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName}, clusterRole)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent clusterRole deployed before, creating new one")
			clusterRole = newClusterRoleForCRD()
			if err := r.Create(ctx, clusterRole); err != nil {
				r.Log.Error(err, "Failed to create Instana agent clusterRole")
			} else {
				r.Log.Info(fmt.Sprintf("%s clusterRole created successfully", AppName))
			}
		}
	}
	return err
}

func (r *InstanaAgentReconciler) reconcileClusterRoleBinding(ctx context.Context) error {
	clusterRoleBinding := &rbacV1.ClusterRoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName}, clusterRoleBinding)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent clusterRoleBinding deployed before, creating new one")
			clusterRoleBinding = newClusterRoleBindingForCRD()
			if err := r.Create(ctx, clusterRoleBinding); err != nil {
				r.Log.Error(err, "Failed to create Instana agent clusterRoleBinding")
			} else {
				r.Log.Info(fmt.Sprintf("%s clusterRoleBinding created successfully", AppName))
			}
		}
	}
	return err
}

func (r *InstanaAgentReconciler) reconcileServiceAccounts(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	serviceAcc := &coreV1.ServiceAccount{}
	err := r.Get(ctx, client.ObjectKey{Name: AgentServiceAccountName, Namespace: AgentNameSpace}, serviceAcc)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent service account deployed before, creating new one")
			serviceAcc = newServiceAccountForCRD()
			if err := r.Create(ctx, serviceAcc); err != nil {
				r.Log.Error(err, "Failed to create service account")
			} else {
				r.Log.Info(fmt.Sprintf("%s service account created successfully", AgentServiceAccountName))
			}
		}
	}
	return err
}

func (r *InstanaAgentReconciler) reconcileServices(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	service := &coreV1.Service{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName, Namespace: AgentNameSpace}, service)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent service deployed before, creating new one")
			service = newServiceForCRD()
			if err := r.Create(ctx, service); err != nil {
				r.Log.Error(err, "Failed to create service")
			} else {
				r.Log.Info(fmt.Sprintf("%s service created successfully", AppName))
			}
		}
	}
	return err
}

func (r *InstanaAgentReconciler) reconcileSecrets(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	secret := &coreV1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: AgentSecretName, Namespace: AgentNameSpace}, secret)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent config secret deployed before, creating new one")
			secret = newSecretForCRD(crdInstance)
			if err := r.Create(ctx, secret); err != nil {
				r.Log.Error(err, "failed to create secret")
			} else {
				r.Log.Info(fmt.Sprintf("%s secret created successfully", AgentSecretName))
			}
		}
	}
	return err
}

func (r *InstanaAgentReconciler) reconcileImagePullSecrets(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	pullSecret := &coreV1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: AgentImagePullSecretName, Namespace: AgentNameSpace}, pullSecret)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent Image pull secret deployed before, creating new one")
			pullSecret := newImagePullSecretForCRD(crdInstance, r.Log)
			if err := r.Create(ctx, pullSecret); err != nil {
				r.Log.Error(err, "Failed to create Image pull secret")
			} else {
				r.Log.Info(fmt.Sprintf("%s image pull secret created successfully", AgentImagePullSecretName))
			}
		}
	}
	return err
}

func (r *InstanaAgentReconciler) reconcileConfigMap(ctx context.Context, crdInstance *instanaV1Beta1.InstanaAgent) error {
	configMap := &coreV1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: AppName, Namespace: AgentNameSpace}, configMap)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.Log.Info("No InstanaAgent configMap deployed before, creating new one")
			configMap := newConfigMapForCRD(crdInstance, r.Log)
			if err := r.Create(ctx, configMap); err != nil {
				r.Log.Error(err, "Failed to create configMap")
			} else {
				r.Log.Info(fmt.Sprintf("%s configMap created successfully", AppName))
			}
		}
	}
	return err
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

// returns a Daemonset object with the data hold in instanaAgent crd instance
func newDaemonsetForCRD(crdInstance *instanaV1Beta1.InstanaAgent) *appV1.DaemonSet {
	//we need to have a same matched label for all our agent resources
	selectorLabels := buildLabels()
	podSpec := newPodSpec(crdInstance)
	return &appV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
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

func newSecretForCRD(crdInstance *instanaV1Beta1.InstanaAgent) *coreV1.Secret {
	return &coreV1.Secret{
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
	}
}

func newImagePullSecretForCRD(crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) *coreV1.Secret {
	return &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AgentImagePullSecretName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Type: coreV1.SecretTypeDockerConfigJson,
		Data: generatePullSecretData(crdInstance, Log),
	}
}

func newConfigMapForCRD(crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) *coreV1.ConfigMap {
	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		},
		Data: map[string]string{
			"cluster_name":       crdInstance.Spec.ClusterName,
			"configuration.yaml": readFile("configuration.yaml", Log),
		},
	}
}

func newServiceForCRD() *coreV1.Service {
	return &coreV1.Service{
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
}

func newServiceAccountForCRD() *coreV1.ServiceAccount {
	return &coreV1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      AppName,
			Namespace: AgentNameSpace,
			Labels:    buildLabels(),
		}}
}

func newClusterRoleForCRD() *rbacV1.ClusterRole {
	return &rbacV1.ClusterRole{
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
}

func newClusterRoleBindingForCRD() *rbacV1.ClusterRoleBinding {
	return &rbacV1.ClusterRoleBinding{
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
}
func (r *InstanaAgentReconciler) fetchCrdInstance(ctx context.Context, req ctrl.Request) (*instanaV1Beta1.InstanaAgent, error) {
	crdInstance := &instanaV1Beta1.InstanaAgent{}
	err := r.Get(ctx, req.NamespacedName, crdInstance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return nil, nil
		}
		// Error reading the object - requeue the request.
		return nil, err
	}
	r.Log.Info("Reconciling Instana CRD")
	AppName = crdInstance.Name
	AgentNameSpace = crdInstance.Namespace
	return crdInstance, err
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
		{Name: "INSTANA_ZONE", Value: crdInstance.Spec.ZoneName},
		{Name: "INSTANA_KUBERNETES_CLUSTER_NAME", Value: crdInstance.Spec.ClusterName},
		{Name: "INSTANA_AGENT_ENDPOINT", Value: crdInstance.Spec.Endpoint.Host},
		{Name: "INSTANA_AGENT_ENDPOINT_PORT", Value: crdInstance.Spec.Endpoint.Port},
		{Name: "INSTANA_AGENT_KEY", Value: crdInstance.Spec.Key},
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
			MountPath: "/DEV",
		},
		{
			Name:      "run",
			MountPath: "/RUN",
		},
		{
			Name:      "var-run",
			MountPath: "/VAR/RUN",
		},
		{
			Name:      "var-run-kubo",
			MountPath: "/VAR/VCAP/SYS/RUN/DOCKER",
		},
		{
			Name:      "sys",
			MountPath: "/SYS",
		},
		{
			Name:      "var-log",
			MountPath: "/VAR/LOG",
		},
		{
			Name:      "var-lib",
			MountPath: "/VAR/LIB/CONTAINERS/STORAGE",
		},
		{
			Name:      "machine-id",
			MountPath: "/ETC/MACHINE-ID",
		},
		{
			Name:      "configuration",
			SubPath:   "configuration.yaml",
			MountPath: "/ROOT/configuration.yaml",
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

func generatePullSecretData(crdInstance *instanaV1Beta1.InstanaAgent, Log logr.Logger) map[string][]byte {
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

func readFile(filename string, Log logr.Logger) string {
	content, err := ioutil.ReadFile("config/" + filename)
	if err != nil {
		Log.Error(err, "failed to read configuration.yaml")
	}
	return string(content)
}
