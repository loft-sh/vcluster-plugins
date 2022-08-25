package serving

const (
	KnativeServiceResourceKind       = "Service"
	KnativeConfigurationResourceKind = "Configuration"

	KnativeServiceName = "hello-ksvc"

	KnativeHelloV1Image = "gcr.io/google-samples/hello-app:1.0"
	KnativeHelloV2Image = "gcr.io/google-samples/hello-app:2.0"

	ServiceVersionV1 = "Version: 1.0.0"
	ServiceBody      = "Hello, world!"
)
