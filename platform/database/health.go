package database

// Ping verifies the shared database handle is usable (for /readyz).
func Ping() error {
	db, err := openShared()
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// PoolStats is a snapshot of the sql.DB pool for metrics gauges.
type PoolStats struct {
	Open    int
	InUse   int
	Idle    int
	Backend string // "sqlite" or "postgres"
}

// Stats returns the current pool snapshot. ok is false when the database
// is not usable (metrics render zeros instead of failing the scrape).
func Stats() (PoolStats, bool) {
	stats := PoolStats{Backend: Backend()}
	db, err := openShared()
	if err != nil {
		return stats, false
	}
	sqlDB, err := db.DB()
	if err != nil {
		return stats, false
	}
	s := sqlDB.Stats()
	stats.Open, stats.InUse, stats.Idle = s.OpenConnections, s.InUse, s.Idle
	return stats, true
}

// Backend reports the active database backend for labels.
func Backend() string {
	if UsingPostgreSQL {
		return "postgres"
	}
	return "sqlite"
}
