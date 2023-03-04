package containers_instana_io_secret

type DockerConfigAuth struct {
	Auth []byte `json:"auth"`
}

type DockerConfigJson struct {
	Auths map[string]DockerConfigAuth `json:"auths"`
}

// Defining this so mock-gen will work
type dockerConfigMarshaler interface {
	MarshalOrDie(obj *DockerConfigJson) []byte
	UnMarshalOrDie(raw []byte) *DockerConfigJson
}
