package containers_instana_io_secret

type DockerConfigAuth struct {
	Auth []byte `json:"auth"`
}

type DockerConfigJson struct {
	Auths map[string]DockerConfigAuth `json:"auths"`
}
