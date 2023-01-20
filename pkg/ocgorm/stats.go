//go:build go1.11
// +build go1.11

package ocgorm

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Tags applied to measures
var (
	// Operation is the type of query (SELECT, INSERT, UPDATE, DELETE)
	Operation, _ = tag.NewKey("sql.operation")

	// Table name of the target database table
	Table, _ = tag.NewKey("sql.table")
)

// Measures
var (
	MeasureQueryCount        = stats.Int64("go.sql/client/calls", "Number of queries started", stats.UnitDimensionless)
	MeasureLatencyMs         = stats.Float64("go.sql/client/latency", "The latency of calls in milliseconds", stats.UnitMilliseconds)
	MeasureOpenConnections   = stats.Int64("go.sql/connections/open", "Count of open connections in the pool", stats.UnitDimensionless)
	MeasureIdleConnections   = stats.Int64("go.sql/connections/idle", "Count of idle connections in the pool", stats.UnitDimensionless)
	MeasureActiveConnections = stats.Int64("go.sql/connections/active", "Count of active connections in the pool", stats.UnitDimensionless)
	MeasureWaitCount         = stats.Int64("go.sql/connections/wait_count", "The total number of connections waited for", stats.UnitDimensionless)
	MeasureWaitDuration      = stats.Float64("go.sql/connections/wait_duration", "The total time blocked waiting for a new connection", stats.UnitMilliseconds)
	MeasureIdleClosed        = stats.Int64("go.sql/connections/idle_closed", "The total number of connections closed due to SetMaxIdleConns", stats.UnitDimensionless)
	MeasureLifetimeClosed    = stats.Int64("go.sql/connections/lifetime_closed", "The total number of connections closed due to SetConnMaxLifetime", stats.UnitDimensionless)
)

// Default distributions used by views in this package
var (
	DefaultMillisecondsDistribution = view.Distribution(
		0.0,
		0.001,
		0.005,
		0.01,
		0.05,
		0.1,
		0.5,
		1.0,
		1.5,
		2.0,
		2.5,
		5.0,
		10.0,
		25.0,
		50.0,
		100.0,
		200.0,
		400.0,
		600.0,
		800.0,
		1000.0,
		1500.0,
		2000.0,
		2500.0,
		5000.0,
		10000.0,
		20000.0,
		40000.0,
		100000.0,
		200000.0,
		500000.0)
)

var (
	SQLClientLatencyView = &view.View{
		Name:        "go.sql/client/latency",
		Description: "The distribution of latencies of various calls in milliseconds",
		Measure:     MeasureLatencyMs,
		Aggregation: DefaultMillisecondsDistribution,
		TagKeys:     []tag.Key{Operation, Table},
	}

	SQLClientCallsView = &view.View{
		Name:        "go.sql/client/calls",
		Description: "The number of various calls of methods",
		Measure:     MeasureQueryCount,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Operation, Table},
	}

	SQLClientOpenConnectionsView = &view.View{
		Name:        "go.sql/db/connections/open",
		Description: "The number of open connections",
		Measure:     MeasureOpenConnections,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	SQLClientIdleConnectionsView = &view.View{
		Name:        "go.sql/db/connections/idle",
		Description: "The number of idle connections",
		Measure:     MeasureIdleConnections,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	SQLClientActiveConnectionsView = &view.View{
		Name:        "go.sql/db/connections/active",
		Description: "The number of active connections",
		Measure:     MeasureActiveConnections,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	SQLClientWaitCountView = &view.View{
		Name:        "go.sql/db/connections/wait_count",
		Description: "The total number of connections waited for",
		Measure:     MeasureWaitCount,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	SQLClientWaitDurationView = &view.View{
		Name:        "go.sql/db/connections/wait_duration",
		Description: "The total time blocked waiting for a new connection",
		Measure:     MeasureWaitDuration,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	SQLClientIdleClosedView = &view.View{
		Name:        "go.sql/db/connections/idle_closed_count",
		Description: "The total number of connections closed due to SetMaxIdleConns",
		Measure:     MeasureIdleClosed,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	SQLClientLifetimeClosedView = &view.View{
		Name:        "go.sql/db/connections/lifetime_closed_count",
		Description: "The total number of connections closed due to SetConnMaxLifetime",
		Measure:     MeasureLifetimeClosed,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	DefaultViews = []*view.View{
		SQLClientCallsView, SQLClientLatencyView, SQLClientOpenConnectionsView,
		SQLClientIdleConnectionsView, SQLClientActiveConnectionsView,
		SQLClientWaitCountView, SQLClientWaitDurationView,
		SQLClientIdleClosedView, SQLClientLifetimeClosedView,
	}
)

// RegisterAllViews registers all ocgorm views to enable collection of stats.
func RegisterAllViews() {
	if err := view.Register(DefaultViews...); err != nil {
		panic(err)
	}
}

// RecordStats records database statistics for provided sql.DB at the provided
// interval. You should defer execution of this function after you establish
// connection to the database `if err == nil { ocgorm.RecordStats(db, 5*time.Second); }
func RecordStats(db *gorm.DB, interval time.Duration) (fnStop func()) {
	var (
		closeOnce sync.Once
		ctx       = context.Background()
		ticker    = time.NewTicker(interval)
		done      = make(chan struct{})
	)

	go func() {
		for {
			select {
			case <-ticker.C:
				dbStats := db.DB().Stats()

				if dbStats.OpenConnections == 0 { // We cleanup the ticker in the event that the database is unavailable
					if err := db.DB().Ping(); strings.Contains(err.Error(), "database is closed") {
						ticker.Stop()
						return
					}
				}

				stats.Record(ctx,
					MeasureOpenConnections.M(int64(dbStats.OpenConnections)),
					MeasureIdleConnections.M(int64(dbStats.Idle)),
					MeasureActiveConnections.M(int64(dbStats.InUse)),
					MeasureWaitCount.M(dbStats.WaitCount),
					MeasureWaitDuration.M(float64(dbStats.WaitDuration.Nanoseconds())/1e6),
					MeasureIdleClosed.M(dbStats.MaxIdleClosed),
					MeasureLifetimeClosed.M(dbStats.MaxLifetimeClosed),
				)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		closeOnce.Do(func() {
			close(done)
		})
	}
}
