package benchmark

import (
	"time"

	"github.com/UniquityVentures/lamu/lamu"
	"github.com/UniquityVentures/lamu/registry"
	"gorm.io/gorm"
)

type Article struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(255);not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

func pluginModels() lamu.PluginFeatures[any] {
	return lamu.PluginFeatures[any]{
		Entries: []registry.Pair[string, any]{
			{Key: "benchmark.Article", Value: Article{}},
		},
	}
}

func pluginDBInitHooks() lamu.PluginFeatures[lamu.DBInitHook] {
	return lamu.PluginFeatures[lamu.DBInitHook]{
		Entries: []registry.Pair[string, lamu.DBInitHook]{
			{
				Key: "benchmark.db_pool",
				Value: func(db *gorm.DB) *gorm.DB {
					sqlDB, err := db.DB()
					if err == nil {
						sqlDB.SetMaxOpenConns(1000)
						sqlDB.SetMaxIdleConns(50)
					}
					return db
				},
			},
		},
	}
}
