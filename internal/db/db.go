package db

import (
	"database/sql"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // import the pure-Go SQLite driver
)

// Open opens a GORM DB connection using pure-Go SQLite.
func Open(path string) (*gorm.DB, error) {
	// Create a database/sql DB using modernc.org/sqlite
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Pass the *sql.DB to GORM
	gdb, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return gdb, nil
}
