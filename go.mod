module github.com/yriveiro/gcs-bucket-operator

go 1.15

require (
	cloud.google.com/go/storage v1.6.0
	github.com/go-logr/logr v0.2.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/prometheus/client_golang v1.1.0 // indirect
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
	sigs.k8s.io/controller-runtime v0.5.0
)
