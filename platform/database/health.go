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
