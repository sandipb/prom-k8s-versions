package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sandipb/prom-k8s-versions/pkg/prom"
	"gopkg.in/alecthomas/kingpin.v2"
)

const appName = "prom-k8s-versions"

var (
	version = "unset"
	commit  = "unset"
	date    = "unset"
)

var (
	showVersion    = kingpin.Flag("version", "Show version").Short('v').Bool()
	debugLevel     = kingpin.Flag("debug", "Debug level logging").Short('d').Bool()
	promServer     = kingpin.Flag("prom-api", "URL to API server").Short('p').Default("localhost:9090").String()
	namespace      = kingpin.Flag("namespace", "Namespace for the app").Short('n').Default("default").String()
	clusters       = kingpin.Flag("clusters", "(Optional) Regex of clusters to select. Can be repeated.").Short('c').Strings()
	timeoutSeconds = kingpin.Flag("timeout", "Timeout in seconds for the query").Short('t').Default("10").Int()
	showPods       = kingpin.Flag("pods", "Show pods").Bool()
	showDeploys    = kingpin.Flag("deploys", "Show deployments,daemonsets and statefulsets").Bool()
	showCms        = kingpin.Flag("config-maps", "Show chart versions for configmaps as well").Bool()
	showAll        bool
)

const helpText = `
Shows a table of pods with their image versions and a table of deployment-like
objects with chart versions.

NOTE: By default, both "--pods" and "--deploys" are implied. But if any one of
them is specified, the other is not shown unless specifically specified.
`

func setup() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.Help = helpText
	kingpin.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	if *debugLevel {
		log.Logger = log.Logger.Level(zerolog.DebugLevel)
	} else {
		log.Logger = log.Logger.Level(zerolog.InfoLevel)
	}

	if !*showPods && !*showDeploys {
		showAll = true
	}
}

func printVersion() {
	fmt.Printf("%s %s, commit %s, built %s\n", appName, version, commit, date)
	os.Exit(0)
}

func main() {
	setup()
	if *showVersion {
		printVersion()
	}

	if !strings.HasPrefix(*promServer, "http") {
		*promServer = "http://" + *promServer
	}

	log.Debug().Msgf("Using prometheus server: %#v", *promServer)
	log.Debug().Msgf("Searching in namespace: %#v", *namespace)

	searchFilter := prom.NewSearchFilter()
	if len(*clusters) > 0 {
		log.Debug().Msgf("Filtering by clusters: %#v", *clusters)
		for _, c := range *clusters {
			searchFilter.AddCluster(c)
		}
	}

	if *showCms {
		searchFilter.IncludeConfigMaps = true
	}

	client := prom.NewClient(*promServer).SetTimeout(*timeoutSeconds)
	data := client.GetInfo(*namespace, searchFilter)

	if data.HasPods() && (showAll || *showPods) {
		printPods(data)
		fmt.Println()
	}

	if data.HasDeploys() && (showAll || *showDeploys) {
		printDeployables(data)
	}
}

func printPods(data prom.ClusterResultSet) {
	fmt.Printf("PODS\n\n")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Cluster", "Pod", "Container", "Image"})
	for _, cluster_name := range data.SortedByCluster() {
		cluster_info := data[cluster_name]
		idx := 0
		for _, e := range cluster_info.Entries {
			// Only handle pods
			if e.Type != prom.EntityPod {
				continue
			}
			idx += 1
			cname := cluster_name
			if idx > 1 {
				cname = ""
			}
			table.Append([]string{cname, e.Name, e.ContainerName, e.ContainerImage})
		}
	}
	table.Render()
}

func printDeployables(data prom.ClusterResultSet) {
	fmt.Printf("DEPLOYS\n\n")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Cluster", "Type", "Name", "Chart"})

	for cluster_name, cluster_info := range data {
		idx := 0
		for _, e := range cluster_info.Entries {
			if e.Type == prom.EntityPod {
				continue
			}
			idx += 1
			cname := cluster_name
			if idx > 1 {
				cname = ""
			}
			table.Append([]string{cname, string(e.Type), e.Name, e.ChartName})
		}
	}
	table.Render()
}
