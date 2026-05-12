package repositories

import (
	"fmt"

	"github.com/jackc/pgconn"
	"gorm.io/gorm"
)

type GormRepository struct {
	DBContext *GormDBContext
}

func (g *GormRepository) SetDBContext(ctx *GormDBContext) {
	g.DBContext = ctx
}

func (g *GormRepository) DB() *gorm.DB {
	return g.DBContext.DB()
}

func isCausedByUniqueConstraint(err error) bool {
	const (
		ErrMySQLDupEntry            = 1062
		ErrMySQLDupEntryWithKeyName = 1586
		ErrPostgresUniqueViolation  = "23505"
	)
	switch sureTypeErr := err.(type) {

	case *pgconn.PgError:
		if sureTypeErr.Code == ErrPostgresUniqueViolation {
			return true
		}
	default:
		return false
	}
	return false
}

type UniqueConstrainError struct {
	OriginalErr error
}

func NewUniqueConstrainError(err error) *UniqueConstrainError {
	return &UniqueConstrainError{
		OriginalErr: err,
	}
}

func (e *UniqueConstrainError) Error() string {
	return "UNIQUE constraint failed"
}

type UniqueConstrainFieldError struct {
	OriginalErr error
	FieldName   string
}

func NewUniqueConstrainFieldError(err error, fieldName string) *UniqueConstrainFieldError {
	return &UniqueConstrainFieldError{
		OriginalErr: err,
		FieldName:   fieldName,
	}
}

func (e *UniqueConstrainFieldError) Error() string {
	return fmt.Sprintf("Field `%s` is not unique", e.FieldName)
}
