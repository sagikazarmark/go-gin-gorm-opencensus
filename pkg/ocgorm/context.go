package ocgorm

import (
	"context"

	"github.com/jinzhu/gorm"
)

// WithContext sets the current context in the db instance for instrumentation.
func WithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.New().Set(contextScopeKey, ctx)
}
