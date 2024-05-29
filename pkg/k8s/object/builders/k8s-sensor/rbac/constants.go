package rbac

const (
	rbacApiGroup   = "rbac.authorization.k8s.io"
	rbacApiVersion = rbacApiGroup + "/v1"
	roleKind       = "ClusterRole"
	subjectKind    = "ServiceAccount"
)
