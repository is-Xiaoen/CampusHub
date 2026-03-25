package svc

import (
	"errors"
	"fmt"
	"log"
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
		return fmt.Errorf("检查 groups.cover_url 字段失败: %w", err)
	}
	if exists {
		log.Printf("[INFO] Chat schema 已包含 groups.cover_url 字段")
		return nil
	}

	if err := db.Exec(addGroupsCoverURLColumnSQL).Error; err != nil {
		if !isDuplicateColumnError(err) {
			return fmt.Errorf("补充 groups.cover_url 字段失败: %w", err)
		}

		exists, checkErr := columnExists(db, "groups", "cover_url")
		if checkErr != nil {
			return fmt.Errorf("复查 groups.cover_url 字段失败: %w", checkErr)
		}
		if exists {
			log.Printf("[INFO] Chat schema 并发修复完成，groups.cover_url 已存在")
			return nil
		}

		return fmt.Errorf("补充 groups.cover_url 字段失败: %w", err)
	}

	log.Printf("[INFO] Chat schema 已修复，补充 groups.cover_url 字段成功")
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
