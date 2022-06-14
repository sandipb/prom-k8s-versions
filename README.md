# prom-k8s-versions

Use prometheus metrics to fetch Kubernetes app version info from various clusters.

It uses the following metrics from [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) to fetch the
version information:

- `kube_pod_container_info`
- `kube_deployment_labels`
- `kube_daemonset_labels`
- `kube_statefulset_labels`

## Usage

```shell-session
$ ./prom-k8s-versions -h
usage: prom-k8s-versions [<flags>]

Shows a table of pods with their image versions and a table of deployment-like objects
with chart versions.

NOTE: By default, both "--pods" and "--deploys" are implied. But if any one of them is
specified, the other is not shown unless specifically specified.

Flags:
  -h, --help                   Show context-sensitive help (also try --help-long and
                               --help-man).
  -v, --version                Show version
  -d, --debug                  Debug level logging
  -p, --prom-api="localhost:9090"  
                               URL to API server
  -n, --namespace="default"    Namespace for the app
  -c, --clusters=CLUSTERS ...  (Optional) Regex of clusters to select. Can be repeated.
  -t, --timeout=10             Timeout in seconds for the query
      --pods                   Show pods
      --deploys                Show deployments,daemonsets and statefulsets
```

## Sample Output

```shell-session
$ ./prom-k8s-versions -p thanos.example.com -n k8s-event-exporter -c dev
PODS

+-----------------+----------------------------------------+--------------------+----------------------------------------+
|     CLUSTER     |                  POD                   |     CONTAINER      |                  IMAGE                 |
+-----------------+----------------------------------------+--------------------+----------------------------------------+
|     dev-1       |    k8s-event-exporter-7cd568dc69-q28dw | k8s-event-exporter | internal/k8s-event-exporter:v0.11-pt.6 |
|     dev-2       |    k8s-event-exporter-7b49ffcb85-zcrjr | k8s-event-exporter | internal/k8s-event-exporter:v0.11-pt.6 |
+-----------------+----------------------------------------+--------------------+----------------------------------------+

DEPLOYS

+-----------------+------------+-----------------------+-----------------------------+
|     CLUSTER     |    TYPE    |         NAME          |            CHART            |
+-----------------+------------+-----------------------+-----------------------------+
|     dev-1       | Deployment |    k8s-event-exporter |    k8s-event-exporter-0.1.6 |
|     dev-2       | Deployment |    k8s-event-exporter |    k8s-event-exporter-0.1.6 |
+-----------------+------------+-----------------------+-----------------------------+
```
