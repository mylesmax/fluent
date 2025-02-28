package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	once sync.Once
)

// id,name,emoji,glow, uuid, etc., tbd
// cur drops
type Profile struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Emoji        string   `json:"emoji"`
	GlowColor    string   `json:"glowColor"`
	CurrentDrops int      `json:"currentDrops"`
	ClassDBPath  string   `json:"classDBPath"`
	ClassUUID    string   `json:"classUUID"`
	IsAddNew     bool     `json:"isAddNew"`
	Active       bool     `json:"active"`
	NameHistory  []string `json:"nameHistory"`
}

// sep into user data/styling, class dbs for droplets, trash can
func ensureDataDirs() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	dirs := []string{
		filepath.Join(homeDir, ".fluent"),
		filepath.Join(homeDir, ".fluent/user"),
		filepath.Join(homeDir, ".fluent/class"),
		filepath.Join(homeDir, ".fluent/trash"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	return nil
}

func createClassDB(classDBPath string, classUUID string, initialName string) error {
	classDB, err := sql.Open("sqlite3", classDBPath)
	if err != nil {
		return err
	}
	defer classDB.Close()

	_, err = classDB.Exec(`
		CREATE TABLE IF NOT EXISTS class_info (
			uuid TEXT PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			name_history TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	nameHistory := []string{initialName}
	nameHistoryJSON, err := json.Marshal(nameHistory)
	if err != nil {
		return err
	}

	_, err = classDB.Exec(`
		INSERT OR REPLACE INTO class_info (uuid, name_history)
		VALUES (?, ?)
	`, classUUID, string(nameHistoryJSON))
	return err
}

func InitDB() (*sql.DB, error) {
	var err error
	once.Do(func() {
		if err = ensureDataDirs(); err != nil {
			log.Printf("Failed to create .fluent directories: %v", err)
			return
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Failed to get home directory: %v", err)
			return
		}

		dbPath := filepath.Join(homeDir, ".fluent", "user", "profiles.db")
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Printf("Failed to open database: %v", err)
			return
		}

		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS profiles (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				emoji TEXT NOT NULL,
				glow_color TEXT NOT NULL,
				current_drops INTEGER DEFAULT 0
			);
		`)
		if err != nil {
			log.Fatal(err)
			return
		}

		columns := []struct {
			name string
			def  string
		}{
			{"class_db_path", "TEXT"},
			{"class_uuid", "TEXT"},
			{"is_add_new", "BOOLEAN DEFAULT 0"},
			{"active", "BOOLEAN DEFAULT 1"},
		}

		for _, col := range columns {
			var exists bool
			err := db.QueryRow(`
				SELECT COUNT(*) > 0 
				FROM pragma_table_info('profiles') 
				WHERE name = ?
			`, col.name).Scan(&exists)

			if err != nil {
				log.Printf("err checkin column %s: %v", col.name, err)
				continue
			}

			if !exists {
				_, err = db.Exec(fmt.Sprintf(`
					ALTER TABLE profiles ADD COLUMN %s %s;
				`, col.name, col.def))
				if err != nil {
					log.Printf("err adding column %s: %v", col.name, err)
				}
			}
		}

		rows, err := db.Query(`
			SELECT id, name 
			FROM profiles 
			WHERE (class_uuid IS NULL OR class_uuid = '') AND name != 'Add New';
		`)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				log.Printf("Error scanning profile: %v", err)
				continue
			}

			classUUID := uuid.New().String()
			classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")

			if err := createClassDB(classDBPath, classUUID, name); err != nil {
				log.Printf("Error creating class DB for %s: %v", name, err)
				continue
			}

			_, err = db.Exec(`
				UPDATE profiles 
				SET class_uuid = ?, class_db_path = ?, active = 1
				WHERE id = ?
			`, classUUID, classDBPath, id)
			if err != nil {
				log.Printf("Error updating profile %s with UUID: %v", name, err)
				continue
			}
		}

		rows2, err := db.Query(`
			SELECT name, class_db_path, class_uuid 
			FROM profiles 
			WHERE class_db_path IS NOT NULL AND name != 'Add New';
		`)
		if err != nil {
			log.Printf("err querying existing profs: %v", err)
			return
		}
		defer rows2.Close()

		for rows2.Next() {
			var name, classDBPath, classUUID string
			if err := rows2.Scan(&name, &classDBPath, &classUUID); err != nil {
				log.Printf("err scanning existing prof: %v", err)
				continue
			}

			classDB, err := sql.Open("sqlite3", classDBPath)
			if err != nil {
				log.Printf("err opening class DB %s: %v", classDBPath, err)
				continue
			}

			_, err = classDB.Exec(`
				CREATE TABLE IF NOT EXISTS class_info (
					uuid TEXT PRIMARY KEY,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					name_history TEXT NOT NULL DEFAULT '[]'
				)
			`)
			if err != nil {
				log.Printf("err creating class_info table: %v", err)
				classDB.Close()
				continue
			}

			//check
			var exists bool
			err = classDB.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM class_info WHERE uuid = ?
				)
			`, classUUID).Scan(&exists)

			if err != nil {
				log.Printf("err checkin UUID existence: %v", err)
				classDB.Close()
				continue
			}

			if !exists {
				//init
				nameHistory := []string{name}
				nameHistoryJSON, _ := json.Marshal(nameHistory)
				_, err = classDB.Exec(`
					INSERT INTO class_info (uuid, name_history)
					VALUES (?, ?)
				`, classUUID, string(nameHistoryJSON))
				if err != nil {
					log.Printf("err initializing class_info: %v", err)
				}
			} else {
				//check
				var nameHistoryJSON string
				err = classDB.QueryRow(`
					SELECT name_history FROM class_info WHERE uuid = ?
				`, classUUID).Scan(&nameHistoryJSON)

				if err != nil || nameHistoryJSON == "" || nameHistoryJSON == "[]" {
					//reset
					nameHistory := []string{name}
					nameHistoryJSON, _ := json.Marshal(nameHistory)
					_, err = classDB.Exec(`
						UPDATE class_info 
						SET name_history = ?
						WHERE uuid = ?
					`, string(nameHistoryJSON), classUUID)
					if err != nil {
						log.Printf("err resetting name_history: %v", err)
					}
				}
			}

			classDB.Close()
		}
	})
	return db, err
}

func GetProfiles() ([]Profile, error) {
	rows, err := db.Query(`
		SELECT id, name, emoji, glow_color, current_drops, class_db_path, class_uuid, is_add_new, active
		FROM profiles 
		WHERE is_add_new = 0 AND active = 1
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		err := rows.Scan(&p.ID, &p.Name, &p.Emoji, &p.GlowColor, &p.CurrentDrops, &p.ClassDBPath, &p.ClassUUID, &p.IsAddNew, &p.Active)
		if err != nil {
			return nil, err
		}

		if p.ClassDBPath != "" {
			classDB, err := sql.Open("sqlite3", p.ClassDBPath)
			if err == nil {
				var nameHistoryJSON string
				err = classDB.QueryRow("SELECT name_history FROM class_info WHERE uuid = ?", p.ClassUUID).Scan(&nameHistoryJSON)
				if err == nil {
					json.Unmarshal([]byte(nameHistoryJSON), &p.NameHistory)
				}
				classDB.Close()
			}
		}

		profiles = append(profiles, p)
	}
	return profiles, nil
}

func AddProfile(profile Profile) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	classUUID := uuid.New().String()
	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")

	if err := createClassDB(classDBPath, classUUID, profile.Name); err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO profiles (name, emoji, glow_color, current_drops, class_db_path, class_uuid, is_add_new, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`, profile.Name, profile.Emoji, profile.GlowColor, profile.CurrentDrops, classDBPath, classUUID, profile.IsAddNew)
	return err
}

func UpdateProfile(profile Profile) error {
	_, err := db.Exec(`
		UPDATE profiles 
		SET name = ?, emoji = ?, glow_color = ?, current_drops = ?
		WHERE id = ?
	`, profile.Name, profile.Emoji, profile.GlowColor, profile.CurrentDrops, profile.ID)
	return err
}

func UpdateProfileName(oldName, newName string) error {
	var classDBPath, classUUID string
	err := db.QueryRow("SELECT class_db_path, class_uuid FROM profiles WHERE name = ?", oldName).Scan(&classDBPath, &classUUID)
	if err != nil {
		return err
	}

	if classDBPath != "" {
		classDB, err := sql.Open("sqlite3", classDBPath)
		if err == nil {
			defer classDB.Close()

			var nameHistoryJSON string
			err = classDB.QueryRow("SELECT name_history FROM class_info WHERE uuid = ?", classUUID).Scan(&nameHistoryJSON)
			if err == nil {
				var nameHistory []string
				json.Unmarshal([]byte(nameHistoryJSON), &nameHistory)
				nameHistory = append(nameHistory, newName)
				newNameHistoryJSON, _ := json.Marshal(nameHistory)

				_, err = classDB.Exec(`
					UPDATE class_info 
					SET name_history = ?, last_modified = CURRENT_TIMESTAMP
					WHERE uuid = ?
				`, string(newNameHistoryJSON), classUUID)
			}
		}
	}

	_, err = db.Exec(`
		UPDATE profiles 
		SET name = ?
		WHERE name = ?
	`, newName, oldName)
	return err
}

func DeleteProfile(name string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	var classDBPath, classUUID string
	err = db.QueryRow("SELECT class_db_path, class_uuid FROM profiles WHERE name = ?", name).Scan(&classDBPath, &classUUID)
	if err != nil {
		return err
	}

	if classDBPath != "" && classUUID != "" {
		trashPath := filepath.Join(homeDir, ".fluent", "trash", classUUID+".db")
		err = os.Rename(classDBPath, trashPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to move class DB to trash: %v", err)
		}
	}

	_, err = db.Exec(`
		UPDATE profiles 
		SET active = 0
		WHERE name = ?
	`, name)
	return err
}

func UpdateDrops(name string, drops int) error {
	_, err := db.Exec(`
		UPDATE profiles 
		SET current_drops = ?
		WHERE name = ?
	`, drops, name)
	return err
}

// try this lter, idgaf it works or not
func RestoreProfile(classUUID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	trashPath := filepath.Join(homeDir, ".fluent", "trash", classUUID+".db")
	classDBPath := filepath.Join(homeDir, ".fluent", "class", classUUID+".db")

	err = os.Rename(trashPath, classDBPath)
	if err != nil {
		return fmt.Errorf("failed to restore class DB from trash: %v", err)
	}

	_, err = db.Exec(`
		UPDATE profiles 
		SET active = 1
		WHERE class_uuid = ?
	`, classUUID)
	return err
}

func UpdateProfileGlowColor(name string, newGlowColor string) error {
	_, err := db.Exec(`
		UPDATE profiles 
		SET glow_color = ?
		WHERE name = ?
	`, newGlowColor, name)
	return err
}
