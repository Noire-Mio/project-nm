package repositories

import (
	"gorm.io/gorm"
)

type GormDBContextFactory func() *GormDBContext

type GormDBContext struct {
	source *gorm.DB
	tx     *gorm.DB
}

func NewGormDBContext(db *gorm.DB) *GormDBContext {
	return &GormDBContext{source: db}
}

func (g *GormDBContext) DB() *gorm.DB {
	if g.tx != nil {
		return g.tx
	} else {
		return g.source
	}
}

func (g *GormDBContext) Begin() error {
	g.tx = g.source.Begin()
	return nil
}

func (g *GormDBContext) Commit() error {
	err := g.tx.Commit().Error
	if err != nil {
		return err
	}
	g.tx = nil
	return nil
}

func (g *GormDBContext) Rollback() {
	g.tx.Rollback()
	g.tx = nil
}

func (g *GormDBContext) Dispose() {
	if g.tx != nil {
		_ = g.tx.Rollback()
		g.tx = nil
	}
}
