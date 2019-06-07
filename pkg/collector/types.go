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
	Database string `db:"database"`
	Table    string `db:"table"`
	Bytes    int64  `db:"bytes"`
	Parts    int64  `db:"parts"`
	Rows     int64  `db:"rows"`
}
