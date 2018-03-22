# promeval

`promeval` is a small CLI tool that allows to test your scrape configuration before
attaching it to the Prometheus instance.

## Setup

- setup your `$GOPATH`
- add `$GOPATH/bin` to your `$PATH`
- get tool:

```
go get github.com/Bplotka/promeval
make install-tools
dep ensure
make build
```

Hardest part:

If you want to run promeval on your local machine you need to "reproduce" setup you have on your Prometheus instance to have proper results.

For example for Kubernetes default SD config you need:
* KUBERNETES_SERVICE_HOST env variable
* KUBERNETES_SERVICE_PORT env variable
* `/var/run/secrets/kubernetes.io/serviceaccount/token`
* `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt`

All of this depends on your Prometheus scraping configuration.

Tool should print properly what is missing in case of missing configuration against certain service discoveries.

## Usage

### Targets

`targets` command allows you to print all discovered jobs after and before relabel.
You can use `--job` and `--source` flags that print only certain sources/jobs.

For example:
`./promeval targets prometheus_config.yaml --job=kubernetes-pods --source=pod/default/service`

will evaluate given prometheus config and print labels that will be used for Kubernetes
pod running in default namespace called "service". E.g:

```
{
	"job_name": "kubernetes-pods",
	"source": "pod/default/service",
	"before": {
		"__address__": "...,
		"__meta_kubernetes_namespace": "default",
		"__meta_kubernetes_pod_annotation_kubernetes_io_created_by": "{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"StatefulSet\",\"namespace\":\"default\",\...",
		"__meta_kubernetes_pod_annotation_pod_alpha_kubernetes_io_initialized": "true",
		"__meta_kubernetes_pod_annotation_pod_beta_kubernetes_io_hostname": "...",
		"__meta_kubernetes_pod_annotation_pod_beta_kubernetes_io_subdomain": "...",
		"__meta_kubernetes_pod_annotation_prometheus_io_path": "/metrics",
		"__meta_kubernetes_pod_annotation_prometheus_io_scheme": "http",
		"__meta_kubernetes_pod_container_name": "service",
		"__meta_kubernetes_pod_container_port_name": "http",
		"__meta_kubernetes_pod_container_port_number": "...",
		"__meta_kubernetes_pod_container_port_protocol": "TCP",
		"__meta_kubernetes_pod_host_ip": "....",
		"__meta_kubernetes_pod_ip": "...",
		"__meta_kubernetes_pod_label_component": "service",
		"__meta_kubernetes_pod_name": "...",
		"__meta_kubernetes_pod_node_name": "....",
		"__meta_kubernetes_pod_ready": "true",
		"__metrics_path__": "/metrics",
		"__scheme__": "http",
		"job": "kubernetes-pods"
	},
	"after": {
		"__address__": "....",
		"__metrics_path__": "/metrics",
		"__scheme__": "http",
		<all resulted label from your relabel process>
	}
},
```

NOTE: It is useful to actually print all before narrowing that to job and source to know exact name for `source` (: It is not straightforward and depends on your job.

Alternatively you can specify prometheus config from Kubernetes configmap directly (named with the key "prometheus.yaml"):

`./promeval targets configmap.yaml --configmap-item=prometheus.yaml --job=kubernetes-pods --source=pod/default/service`

### relabel

WIP: Basic idea is to feed labels that you are interested in manually and do relabel on it using given relabeling from your config.
