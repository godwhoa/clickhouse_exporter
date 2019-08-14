package collector

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// Prometheus metrics namespace
const metricsNamespace = "clickhouse"

// ClickHouseCollector collects clickhouse metrics.
type ClickHouseCollector struct {
	db *sqlx.DB
}

// NewClickHouseCollector returns an initialized ClickHouseCollector.
func NewClickHouseCollector(db *sqlx.DB) *ClickHouseCollector {
	return &ClickHouseCollector{db}
}

// Describe describes all the metrics.
func (c *ClickHouseCollector) Describe(ch chan<- *prometheus.Desc) {
	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	c.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

// Collect fetches the metrics from clickhouse.
func (c *ClickHouseCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.collect(ch); err != nil {
		log.Errorf("failed to collect clickhouse metrics: %s", err)
	}
}

func (c *ClickHouseCollector) collect(ch chan<- prometheus.Metric) error {
	var metrics []Metric
	err := c.db.Select(&metrics, "SELECT metric, value FROM system.metrics")
	if err != nil {
		return err
	}
	for _, m := range metrics {
		vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      fixMetricName(m.Metric),
			Help:      fmt.Sprintf("Number of %s currently processed", m.Metric),
		}, []string{}).WithLabelValues()
		vec.Set(m.Value)
		vec.Collect(ch)
	}

	var asyncMetrics []Metric
	err = c.db.Select(&asyncMetrics, "SELECT metric, value FROM system.asynchronous_metrics")
	if err != nil {
		return err
	}
	for _, m := range asyncMetrics {
		vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      fixMetricName(m.Metric),
			Help:      fmt.Sprintf("Number of %s async processed", m.Metric),
		}, []string{}).WithLabelValues()
		vec.Set(m.Value)
		vec.Collect(ch)
	}

	var events []Event
	err = c.db.Select(&events, "SELECT event, value FROM system.events")
	if err != nil {
		return err
	}
	for _, e := range events {
		m, err := prometheus.NewConstMetric(
			prometheus.NewDesc(
				fmt.Sprintf("%s_%s_total", metricsNamespace, fixMetricName(e.Event)),
				fmt.Sprintf("Number of %s total processed", e.Event),
				[]string{}, nil),
			prometheus.CounterValue, e.Value)
		if err != nil {
			log.Errorf("failed to init const metric: %v", err)
			continue
		}
		ch <- m
	}

	var partMetrics []Part
	err = c.db.Select(&partMetrics, `SELECT count() AS parts, partition, database, active, table  FROM system.parts GROUP BY partition, database, table, active`)
	if err != nil {
		return err
	}
	for _, part := range partMetrics {
		if part.Active == 1 {
			countVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: metricsNamespace,
				Name:      "active_parts_count",
				Help:      "Number of parts of the table",
			}, []string{"database", "table", "partition"}).WithLabelValues(part.Database, part.Table, part.Partition)
			countVec.Set(float64(part.Parts))
			countVec.Collect(ch)
		} else {
			countVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: metricsNamespace,
				Name:      "inactive_parts_count",
				Help:      "Number of parts of the table",
			}, []string{"database", "table", "partition"}).WithLabelValues(part.Database, part.Table, part.Partition)
			countVec.Set(float64(part.Parts))
			countVec.Collect(ch)
		}
	}

	var replicaMetrics []Replica
	err = c.db.Select(&replicaMetrics, `
	SELECT 
		database, 
		table,
		is_readonly, 
		future_parts, 
		parts_to_check, 
		inserts_in_queue,
		merges_in_queue, 
		total_replicas, 
		active_replicas
	FROM system.replicas
	`)
	if err != nil {
		return err
	}

	for _, replica := range replicaMetrics {
		isReadOnly := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "replica_readonly",
			Help:      "Readonly only status of a replica",
		}, []string{"database", "table"}).WithLabelValues(replica.Database, replica.Table)
		isReadOnly.Set(float64(replica.ReadOnly))
		isReadOnly.Collect(ch)

		futureParts := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "replica_future_parts",
			Help:      "The number of data parts that will appear as the result of INSERTs or merges that haven't been done yet",
		}, []string{"database", "table"}).WithLabelValues(replica.Database, replica.Table)
		futureParts.Set(float64(replica.FutureParts))
		futureParts.Collect(ch)

		checkParts := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "replica_parts_to_check",
			Help:      "The number of data parts in the queue for verification",
		}, []string{"database", "table"}).WithLabelValues(replica.Database, replica.Table)
		checkParts.Set(float64(replica.PartsToCheck))
		checkParts.Collect(ch)

		insertQueue := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "replica_inserts_in_queue",
			Help:      "Number of inserts of blocks of data that need to be made",
		}, []string{"database", "table"}).WithLabelValues(replica.Database, replica.Table)
		insertQueue.Set(float64(replica.InsertsInQueue))
		insertQueue.Collect(ch)

		mergesQueue := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "replica_merges_in_queue",
			Help:      "The number of merges waiting to be made",
		}, []string{"database", "table"}).WithLabelValues(replica.Database, replica.Table)
		mergesQueue.Set(float64(replica.MergesInQueue))
		mergesQueue.Collect(ch)

		totalReplicas := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "replicas_total",
			Help:      "The total number of known replicas of this table",
		}, []string{"database", "table"}).WithLabelValues(replica.Database, replica.Table)
		totalReplicas.Set(float64(replica.TotalReplicas))
		totalReplicas.Collect(ch)

		activeReplicas := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "active_replicas_total",
			Help:      "The number of replicas of this table that have a session in ZooKeeper",
		}, []string{"database", "table"}).WithLabelValues(replica.Database, replica.Table)
		activeReplicas.Set(float64(replica.ActiveReplicas))
		activeReplicas.Collect(ch)

	}
	return nil
}

var _ prometheus.Collector = (*ClickHouseCollector)(nil)
