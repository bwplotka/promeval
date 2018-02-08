# promeval

`promeval` is a small CLI tool that allows to test your scrape configuration before
attaching it to the Prometheus instance.

## Setup

Set your `$GOPATH` and add to your `$GOPATH/bin` `$PATH`
Get tool:

```
go get github.com/Bplotka/promeval
make install-tools
dep ensure
make build
```

Hardest part:

If you really want to run promeval on your local machine you need to "reproduce" setup you have on your Prometheus instance to have proper results.

For example for Kubernetes default SD config you need:
* KUBERNETES_SERVICE_HOST env variable
* KUBERNETES_SERVICE_PORT env variable
* `/var/run/secrets/kubernetes.io/serviceaccount/token`
* `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt`

All of this depends on your Prometheus scraping configuration.

## Usage

Using prometheus config file:

`./promeval <cmd> prometheus.yaml`

Using prometheus config file inside Kubernetes configmap (named with the key "prometheus.yaml"):

`./promeval <cmd> configmap.yaml --configmap-item=prometheus.yaml`

For example:
`./promeval targets prometheus.yaml -job=xxx` will print you labels for targets in the time of scrape (before relabel)
