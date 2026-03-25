package svc

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

const (
	groupsCoverURLColumnExistsSQL = "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?"
	addGroupsCoverURLColumnSQL    = "ALTER TABLE `groups` ADD COLUMN `cover_url` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '封面图URL（活动封面）' AFTER `name`"
)

func ensureChatSchema(db *gorm.DB) error {
	exists, err := columnExists(db, "groups", "cover_url")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if err := db.Exec(addGroupsCoverURLColumnSQL).Error; err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}

		exists, checkErr := columnExists(db, "groups", "cover_url")
		if checkErr != nil {
			return checkErr
		}
		if exists {
			return nil
		}

		return err
	}

	return nil
}

func columnExists(db *gorm.DB, tableName string, columnName string) (bool, error) {
	var count int64
	if err := db.Raw(groupsCoverURLColumnExistsSQL, tableName, columnName).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}

	errText := err.Error()
	return strings.Contains(errText, "Duplicate column name") ||
		strings.Contains(errText, "Error 1060") ||
		errors.Is(err, gorm.ErrDuplicatedKey)
}
