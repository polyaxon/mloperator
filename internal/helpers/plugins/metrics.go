package plugins

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"

	apiv1 "github.com/polyaxon/mloperator/api/v1"
)

// Metrics exposed by the operator
type Metrics struct {
	cli                   client.Client
	JobsRunning           *prometheus.GaugeVec
	JobsCreated           *prometheus.CounterVec
	JobsFailedCreated     *prometheus.CounterVec
	ServicesRunning       *prometheus.GaugeVec
	ServicesCreated       *prometheus.CounterVec
	ServicesFailedCreated *prometheus.CounterVec
	ClustersRunning       *prometheus.GaugeVec
	ClustersCreated       *prometheus.CounterVec
	ClustersFailedCreated *prometheus.CounterVec
	KfJobsRunning         *prometheus.GaugeVec
	KfJobsCreated         *prometheus.CounterVec
	KfJobsFailedCreated   *prometheus.CounterVec
}

// NewMetrics prometheus initializer
func NewMetrics(cli client.Client) *Metrics {
	m := &Metrics{
		cli: cli,
		JobsRunning: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "jobs_running",
				Help: "Current running jobs in the cluster",
			},
			[]string{"namespace"},
		),
		JobsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_created",
				Help: "Total number of jobs created",
			},
			[]string{"namespace"},
		),
		JobsFailedCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_create_failed_total",
				Help: "Total number of jobs creation failures",
			},
			[]string{"namespace"},
		),
		ServicesRunning: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "services_running",
				Help: "Current running services in the cluster",
			},
			[]string{"namespace"},
		),
		ServicesCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "services_created",
				Help: "Total number of services created",
			},
			[]string{"namespace"},
		),
		ServicesFailedCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "services_create_failed_total",
				Help: "Total number of services creation failures",
			},
			[]string{"namespace"},
		),
		ClustersRunning: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "clusters_running",
				Help: "Current running clusters in the cluster",
			},
			[]string{"namespace"},
		),
		ClustersCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "clusters_created",
				Help: "Total number of clusters created",
			},
			[]string{"namespace"},
		),
		ClustersFailedCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "clusters_create_failed_total",
				Help: "Total number of clusters creation failures",
			},
			[]string{"namespace"},
		),
		KfJobsRunning: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kfjobs_running",
				Help: "Current running kfjobs in the cluster",
			},
			[]string{"namespace"},
		),
		KfJobsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kfjobs_created",
				Help: "Total number of kfjobs created",
			},
			[]string{"namespace"},
		),
		KfJobsFailedCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kfjobs_create_failed_total",
				Help: "Total number of kfjobs creation failures",
			},
			[]string{"namespace"},
		),
	}

	metrics.Registry.MustRegister(m)
	return m
}

// Describe implements the prometheus.Collector interface.
func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.JobsRunning.Describe(ch)
	m.JobsCreated.Describe(ch)
	m.JobsFailedCreated.Describe(ch)
	m.ServicesRunning.Describe(ch)
	m.ServicesCreated.Describe(ch)
	m.ServicesFailedCreated.Describe(ch)
	m.ClustersRunning.Describe(ch)
	m.ClustersCreated.Describe(ch)
	m.ClustersFailedCreated.Describe(ch)
	m.KfJobsRunning.Describe(ch)
	m.KfJobsCreated.Describe(ch)
	m.KfJobsFailedCreated.Describe(ch)
}

// Collect implements the prometheus.Collector interface.
func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.scrape()
	m.JobsRunning.Collect(ch)
	m.JobsCreated.Collect(ch)
	m.JobsFailedCreated.Collect(ch)
	m.ServicesRunning.Collect(ch)
	m.ServicesCreated.Collect(ch)
	m.ServicesFailedCreated.Collect(ch)
	m.ClustersRunning.Collect(ch)
	m.ClustersCreated.Collect(ch)
	m.ClustersFailedCreated.Collect(ch)
	m.KfJobsRunning.Collect(ch)
	m.KfJobsCreated.Collect(ch)
	m.KfJobsFailedCreated.Collect(ch)
}

// scrape gets current running jobs.
func (m *Metrics) scrapeJob() {
	jobs := &apiv1.JobList{}
	err := m.cli.List(context.TODO(), jobs)
	if err != nil {
		return
	}
	stsCache := make(map[string]float64)
	for _, job := range jobs.Items {
		stsCache[job.Namespace]++
	}

	for ns, op := range stsCache {
		m.JobsRunning.WithLabelValues(ns).Set(op)
	}
}

// scrape gets current running services.
func (m *Metrics) scrapeService() {
	services := &apiv1.ServiceList{}
	err := m.cli.List(context.TODO(), services)
	if err != nil {
		return
	}
	stsCache := make(map[string]float64)
	for _, service := range services.Items {
		stsCache[service.Namespace]++
	}

	for ns, op := range stsCache {
		m.ServicesRunning.WithLabelValues(ns).Set(op)
	}
}

// scrape gets current running clusters.
func (m *Metrics) scrapeCluster() {
	clusters := &apiv1.ClusterList{}
	err := m.cli.List(context.TODO(), clusters)
	if err != nil {
		return
	}
	stsCache := make(map[string]float64)
	for _, cluster := range clusters.Items {
		stsCache[cluster.Namespace]++
	}

	for ns, op := range stsCache {
		m.ClustersRunning.WithLabelValues(ns).Set(op)
	}
}

// scrape gets current running kfjobs.
func (m *Metrics) scrapeKfJob() {
	kfjobs := &apiv1.KfJobList{}
	err := m.cli.List(context.TODO(), kfjobs)
	if err != nil {
		return
	}
	stsCache := make(map[string]float64)
	for _, kfjob := range kfjobs.Items {
		stsCache[kfjob.Namespace]++
	}

	for ns, op := range stsCache {
		m.KfJobsRunning.WithLabelValues(ns).Set(op)
	}
}

// scrape all resources.
func (m *Metrics) scrape() {
	m.scrapeJob()
	m.scrapeService()
	m.scrapeCluster()
	m.scrapeKfJob()
}
