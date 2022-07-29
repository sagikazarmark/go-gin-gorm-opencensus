package ocgorm

// Attributes recorded on the span for the queries.
const (
	// Datadog expects the query text here to enable aggregations of queries
	// Must be used in tandem with a service.name and span.type attribute
	// Our fork uses this instead of gorm.query
	ResourceNameAttribute = "resource.name"

	TableAttribute = "gorm.table"
)
