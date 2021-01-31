module github.com/yriveiro/gcs-bucket-operator

go 1.15

require (
	cloud.google.com/go v0.38.0
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/prometheus/client_golang v1.1.0 // indirect
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449 // indirect
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)
