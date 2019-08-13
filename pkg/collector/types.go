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
	Active    int64  `db:"active"`
	Parts     int64  `db:"parts"`
}
