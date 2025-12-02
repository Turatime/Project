// database/bootstrap.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
	sqlite "github.com/glebarez/sqlite" // CGO-free driver; keep your current if you prefer

	"aoi/entities"
)

func OpenSQLite(path string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}

	// IMPORTANT: run the PK migration BEFORE AutoMigrate so GORM doesn't try the failing ALTER TABLE
	if err := migrateReplanLogsAddPK(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// your other automigrates...
	if err := db.AutoMigrate(
		&entities.Field{},
		&entities.Plan{},
		&entities.ScheduleTask{},
		&entities.ReplanLog{}, // now safe: table already has PK
		&entities.KBDocument{},
		&entities.KBChunk{},
	); err != nil {
		log.Fatalf("automigrate: %v", err)
	}

	return db
}

// migrateReplanLogsAddPK rebuilds replan_logs if it lacks a primary key on id.
func migrateReplanLogsAddPK(db *gorm.DB) error {
	// does table exist?
	var tbl string
	if err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name='replan_logs'`).Scan(&tbl).Error; err != nil {
		return fmt.Errorf("check table exist: %w", err)
	}
	if tbl == "" {
		// fresh DB, nothing to do
		return nil
	}

	// inspect columns
	type colInfo struct {
		Cid       int
		Name      string
		Type      string
		NotNull   int
		DfltValue sql.NullString
		Pk        int
	}
	var cols []colInfo
	if err := db.Raw(`PRAGMA table_info(replan_logs)`).Scan(&cols).Error; err != nil {
		return fmt.Errorf("table_info: %w", err)
	}

	hasIDasPK := false
	lower := func(s string) string { return strings.ToLower(s) }
	for _, c := range cols {
		if lower(c.Name) == "id" {
			if c.Pk == 1 {
				hasIDasPK = true
			}
			break
		}
	}
	if hasIDasPK {
		// already good
		return nil
	}

	// We must rebuild the table.
	// Adjust the target schema to match entities.ReplanLog (and your current columns).
	createSQL := `
CREATE TABLE replan_logs_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    field_id INTEGER,
    plan_id INTEGER,
    reason TEXT,
    delta_md TEXT,
    problems TEXT,      -- JSON text (gorm serializer)
    created_at DATETIME
);
`
	// figure which columns exist in old table
	oldCols := map[string]bool{}
	for _, c := range cols {
		oldCols[lower(c.Name)] = true
	}
	sel := func(name string) string {
		if oldCols[name] {
			return name
		}
		return "NULL AS " + name
	}
	copySQL := fmt.Sprintf(`
INSERT INTO replan_logs_new (field_id, plan_id, reason, delta_md, problems, created_at)
SELECT %s, %s, %s, %s, %s, %s FROM replan_logs;
`,
		sel("field_id"),
		sel("plan_id"),
		sel("reason"),
		sel("delta_md"),
		sel("problems"),
		sel("created_at"),
	)

	// do it in a transaction
	return db.Transaction(func(tx *gorm.DB) error {
		// turn off FK checks during rebuild (SQLite scopes to connection; OK for our short tx)
		if err := tx.Exec(`PRAGMA foreign_keys=OFF`).Error; err != nil {
			return err
		}
		if err := tx.Exec(createSQL).Error; err != nil {
			return err
		}
		if err := tx.Exec(copySQL).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DROP TABLE replan_logs`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`ALTER TABLE replan_logs_new RENAME TO replan_logs`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`PRAGMA foreign_keys=ON`).Error; err != nil {
			return err
		}
		return nil
	})
}
