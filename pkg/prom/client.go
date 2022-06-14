package prom

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/rs/zerolog/log"
)

const defaultTimeout = 10 * time.Second

type PromClient struct {
	client   promapi.Client
	v1client promapiv1.API
	timeout  time.Duration
}

func NewClient(url string) *PromClient {

	client, err := promapi.NewClient(promapi.Config{
		Address: url,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create prometheus client")
	}

	return &PromClient{
		client:   client,
		v1client: promapiv1.NewAPI(client),
		timeout:  defaultTimeout,
	}

}

func (pc *PromClient) SetTimeout(seconds int) *PromClient {
	pc.timeout = time.Second * time.Duration(seconds)
	return pc
}

func (pc *PromClient) GetInfo(ns string, filter ClusterFilter) ClusterResultSet {
	query := fmt.Sprintf(`{namespace="%s", __name__=~'kube_pod_container_info|kube_(deployment|daemonset|statefulset)_labels'}`, ns)
	log.Debug().Msgf("Using prom query: %s", query)

	ctx, cancel := context.WithTimeout(context.Background(), pc.timeout)
	defer cancel()

	result, warnings, err := pc.v1client.Query(ctx, query, time.Now())
	if err != nil {
		log.Fatal().Err(err).Msg("Error querying Prometheus")
	}
	if len(warnings) > 0 {
		log.Warn().Msgf("Prometheus query warnings: %#v", warnings)
	}
	if result.Type() != model.ValVector {
		log.Fatal().Msgf("Unexpected non-vector result type %s received for query: %s", result.Type(), query)
	}
	results := result.(model.Vector)
	log.Debug().Msgf("%d metrics received", len(results))

	out := map[string]*ClusterEntry{}

	for _, entry := range results {
		labels := MetricLabels(entry.Metric)

		clusterName := labels["cluster_name"]

		// apply filter
		if !filter.Has(clusterName) {
			log.Debug().Msgf("Skipping result for %s as it does not match filter", clusterName)
			continue
		}

		clusterEntry, ok := out[clusterName]
		if !ok {
			clusterEntry = NewClusterEntry(clusterName)
			out[clusterName] = clusterEntry
		}

		var entry EntityInfo

		metricName := labels["__name__"]
		switch metricName {
		case "kube_pod_container_info":
			entry.Type = EntityPod
			entry.ContainerInfo = ContainerInfo{
				ContainerName:  labels["container"],
				ContainerImage: labels["image"],
			}
			entry.Name = labels["pod"]
			entry.ContainerImage = strings.Replace(entry.ContainerImage, "docker.io/", "", -1)

		// kube_(deployment|daemonset|statefulset)_labels
		case "kube_deployment_labels":
			entry.Type = EntityDeployment
			entry.Name = labels["deployment"]
		case "kube_daemonset_labels":
			entry.Type = EntityDaemonset
			entry.Name = labels["statefulset"]
		case "kube_statefulset_labels":
			entry.Type = EntityStatefulset
			entry.Name = labels["daemonset"]
			fmt.Printf("%#v\n", entry)
		default:
			log.Warn().Msgf("Ignoring unexpected metric name: %s", metricName)
			continue
		} // switch
		entry.SetChartFrom(labels)
		clusterEntry.Entries = append(clusterEntry.Entries, entry)
	}

	for k := range out {
		sort.Sort(EntityInfoList(out[k].Entries))
	}
	return out
}

type ContainerInfo struct {
	ContainerName  string
	ContainerImage string
}

type EntityType string

const (
	EntityDeployment  EntityType = "Deployment"
	EntityDaemonset   EntityType = "DaemonSet"
	EntityStatefulset EntityType = "StatefulSet"
	EntityPod         EntityType = "Pod"
)

type EntityInfo struct {
	Name      string
	Type      EntityType
	ChartName string
	ContainerInfo
}

func (e *EntityInfo) SetChartFrom(labels map[string]string) {
	if name, ok := labels["label_chart"]; ok {
		e.ChartName = name
	} else if name, ok := labels["label_helm_sh_chart"]; ok {
		e.ChartName = name
	}
}

type EntityInfoList []EntityInfo

// sort.Interface implementation

func (eil EntityInfoList) Len() int {
	return len(eil)
}

func (eil EntityInfoList) Swap(i, j int) {
	eil[i], eil[j] = eil[j], eil[i]
}
func (eil EntityInfoList) Less(i, j int) bool {
	return (eil[i].Type < eil[j].Type) && (eil[i].Name < eil[j].Name)
}

type ClusterEntry struct {
	ClusterName string
	Entries     []EntityInfo
}

func NewClusterEntry(name string) *ClusterEntry {
	return &ClusterEntry{
		ClusterName: name,
		Entries:     []EntityInfo{},
	}
}

type ClusterResultSet map[string]*ClusterEntry

// sortedClusters returns a sorted list of Cluster names
func (crs ClusterResultSet) SortedByCluster() []string {
	clusters := make([]string, 0, len(crs))
	for k := range crs {
		clusters = append(clusters, k)
	}
	sort.Strings(clusters)
	return clusters
}

// HasPods returns true if the result set has any pods to display
func (crs ClusterResultSet) HasPods() bool {
	for cname := range crs {
		for _, e := range crs[cname].Entries {
			if e.Type == EntityPod {
				return true
			}
		}
	}
	return false
}

// HasDeploys returns true if the result set has any deployables to display
func (crs ClusterResultSet) HasDeploys() bool {
	for cname := range crs {
		for _, e := range crs[cname].Entries {
			if e.Type != EntityPod {
				return true
			}
		}
	}
	return false
}

func MetricLabels(m model.Metric) map[string]string {
	labelset := model.LabelSet(m)
	labels := map[string]string{}
	for l, v := range labelset {
		labels[string(l)] = string(v)
	}
	return labels
}

type ClusterFilter map[string]*regexp.Regexp

func (cf ClusterFilter) Empty() bool {
	return len(cf) == 0
}

func (cf ClusterFilter) Add(name string) {
	cf[name] = regexp.MustCompile(name)
}

// Has return True if there are no filters, or if there is at least one filter which matches name
func (cf ClusterFilter) Has(name string) bool {
	if cf.Empty() {
		return true
	}

	for _, pattern := range cf {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}
