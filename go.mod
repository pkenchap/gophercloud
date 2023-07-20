module github.com/gophercloud/gophercloud

go 1.20

require (
	golang.org/x/crypto v0.11.0
	gopkg.in/yaml.v2 v2.4.0
  go.opentelemetry.io/contrib/propagators v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
)

require golang.org/x/sys v0.10.0 // indirect
