package svc

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	columnExistsQuery = "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?"
	addCoverURLSQL    = "ALTER TABLE `groups` ADD COLUMN `cover_url` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '封面图URL（活动封面）' AFTER `name`"
)

func TestEnsureChatSchemaSkipsAlterWhenCoverURLExists(t *testing.T) {
	db, mock := newMockGormDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(columnExistsQuery)).
		WithArgs("groups", "cover_url").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	if err := ensureChatSchema(db); err != nil {
		t.Fatalf("ensureChatSchema returned error: %v", err)
	}

	assertExpectations(t, mock)
}

func TestEnsureChatSchemaAddsCoverURLWhenMissing(t *testing.T) {
	db, mock := newMockGormDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(columnExistsQuery)).
		WithArgs("groups", "cover_url").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(addCoverURLSQL)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := ensureChatSchema(db); err != nil {
		t.Fatalf("ensureChatSchema returned error: %v", err)
	}

	assertExpectations(t, mock)
}

func TestEnsureChatSchemaReturnsMetadataQueryError(t *testing.T) {
	db, mock := newMockGormDB(t)
	expectedErr := errors.New("metadata query failed")

	mock.ExpectQuery(regexp.QuoteMeta(columnExistsQuery)).
		WithArgs("groups", "cover_url").
		WillReturnError(expectedErr)

	err := ensureChatSchema(db)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	assertExpectations(t, mock)
}

func TestEnsureChatSchemaReturnsAlterError(t *testing.T) {
	db, mock := newMockGormDB(t)
	expectedErr := errors.New("alter table failed")

	mock.ExpectQuery(regexp.QuoteMeta(columnExistsQuery)).
		WithArgs("groups", "cover_url").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(addCoverURLSQL)).
		WillReturnError(expectedErr)

	err := ensureChatSchema(db)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	assertExpectations(t, mock)
}

func TestEnsureChatSchemaTreatsDuplicateColumnAsSuccess(t *testing.T) {
	db, mock := newMockGormDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(columnExistsQuery)).
		WithArgs("groups", "cover_url").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(addCoverURLSQL)).
		WillReturnError(errors.New("Error 1060 (42S21): Duplicate column name 'cover_url'"))
	mock.ExpectQuery(regexp.QuoteMeta(columnExistsQuery)).
		WithArgs("groups", "cover_url").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	if err := ensureChatSchema(db); err != nil {
		t.Fatalf("ensureChatSchema returned error: %v", err)
	}

	assertExpectations(t, mock)
}

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open failed: %v", err)
	}

	return db, mock
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
