package helpers

type OpenTelemetrySettings interface {
	GrpcIsEnabled() bool
	HttpIsEnabled() bool
	IsEnabled() bool
}
