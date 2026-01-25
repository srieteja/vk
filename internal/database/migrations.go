package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type appliedMigration struct {
	Name      string    `gorm:"column:name;primaryKey"`
	AppliedAt time.Time `gorm:"column:applied_at"`
}

func (appliedMigration) TableName() string {
	return "schema_migrations"
}

func ApplyMigrations(db *gorm.DB, dir string) error {
	if dir == "" {
		return fmt.Errorf("migrations dir is empty")
	}

	if err := db.AutoMigrate(&appliedMigration{}); err != nil {
		return fmt.Errorf("failed to prepare schema_migrations: %w", err)
	}

	applied := map[string]struct{}{}
	var rows []appliedMigration
	if err := db.Find(&rows).Error; err != nil {
		return fmt.Errorf("failed to load applied migrations: %w", err)
	}
	for _, row := range rows {
		applied[row.Name] = struct{}{}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if name == "init.sql" {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}

	sort.Strings(files)

	for _, file := range files {
		base := filepath.Base(file)
		if _, ok := applied[base]; ok {
			continue
		}

		body, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", base, err)
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(body)).Error; err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", base, err)
			}
			if err := tx.Create(&appliedMigration{Name: base, AppliedAt: time.Now()}).Error; err != nil {
				return fmt.Errorf("failed to record migration %s: %w", base, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}
