package collector

// Metric from ClickHouse.
type Metric struct {
	Metric string  `db:"metric"`
	Value  float64 `db:"value"`
}

// Event from ClickHouse.
type Event struct {
	Event string  `db:"event"`
	Value float64 `db:"value"`
}

// Part metrics from ClickHouse
type Part struct {
	Database  string `db:"database"`
	Table     string `db:"table"`
	Partition string `db:"partition"`
	Active    uint8  `db:"active"`
	Parts     int64  `db:"parts"`
}

type Replica struct {
	Database       string `db:"database"`
	Table          string `db:"table"`
	ReadOnly       uint8  `db:"is_readonly"`
	FutureParts    int64  `db:"future_parts"`
	PartsToCheck   int64  `db:"parts_to_check"`
	InsertsInQueue int64  `db:"inserts_in_queue"`
	MergesInQueue  int64  `db:"merges_in_queue"`
	TotalReplicas  int64  `db:"total_replicas"`
	ActiveReplicas int64  `db:"active_replicas"`
}
