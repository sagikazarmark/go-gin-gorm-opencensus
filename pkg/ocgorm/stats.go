package ocgorm

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Measures
var (
	QueryCount = stats.Int64("opencensus.io/gorm/query_count", "Number of queries started", stats.UnitDimensionless)
)

// Tags applied to measures
var (
	// Operation is the type of query (SELECT, INSERT, UPDATE, DELETE)
	Operation, _ = tag.NewKey("gorm.operation")

	// Table name of the target database table
	Table, _ = tag.NewKey("gorm.table")
)

var (
	QueryCountView = &view.View{
		Name:        "opencensus.io/gorm/query_count",
		Description: "Count of queries started",
		TagKeys:     []tag.Key{Operation, Table},
		Measure:     QueryCount,
		Aggregation: view.Count(),
	}
)
