package db

import (
	"log"
	"src/models"

	"gorm.io/gorm"
)

func CreateTables(db *gorm.DB) {
	db.Exec("SET TIME ZONE 'Asia/Bangkok'")

	err := db.AutoMigrate(
		&models.Config{},
		&models.Subscription{},
		&models.Counter{},
		&models.User{},
		&models.Topic{},
		&models.CounterTopic{},
		&models.Queue{},
		&models.Feedback{},
		&models.NotiSchedule{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate models: %v", err)
	} else {
		log.Println("Successfully migrated tables")
	}

	// ResetSequences(db)
}

func ResetSequences(db *gorm.DB) {
	resetSequenceQuery := `
		DO $$
		DECLARE
			seq_name text;
			table_name text;
			max_id bigint;
		BEGIN
			FOR table_name IN 
				SELECT columns.table_name
				FROM information_schema.columns AS columns
				WHERE columns.column_default LIKE 'nextval%' AND columns.table_schema = 'public' 
			LOOP
				EXECUTE format('SELECT COALESCE(MAX(id), 1) FROM %I', table_name) INTO max_id;
				EXECUTE format(
					'SELECT setval(pg_get_serial_sequence(''%I'', ''id''), %s, false)',
					table_name, max_id
				);
				RAISE NOTICE 'Resetting sequence for table: % with max_id: %', table_name, max_id;
			END LOOP;
		END $$;
	`
	if err := db.Exec(resetSequenceQuery).Error; err != nil {
		log.Fatalf("Failed to reset sequences: %v", err)
	} else {
		log.Println("Successfully reset sequences for all tables.")
	}
}
