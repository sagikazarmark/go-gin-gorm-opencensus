package ocgorm

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
)

// Gorm scope keys
var (
	contextScopeKey = "_opencensusContext"
	spanScopeKey    = "_opencensusSpan"
)

// Option allows for managing ocgorm configuration using functional options.
type Option interface {
	apply(c *callbacks)
}

// OptionFunc converts a regular function to an Option if it's definition is compatible.
type OptionFunc func(c *callbacks)

func (fn OptionFunc) apply(c *callbacks) {
	fn(c)
}

// AllowRoot allows creating root spans in the absence of existing spans.
type AllowRoot bool

func (a AllowRoot) apply(c *callbacks) {
	c.allowRoot = bool(a)
}

// Query allows recording the sql queries in spans.
type Query bool

func (q Query) apply(c *callbacks) {
	c.query = bool(q)
}

// StartOptions configures the initial options applied to a span.
func StartOptions(o trace.StartOptions) Option {
	return OptionFunc(func(c *callbacks) {
		c.startOptions = o
	})
}

// DefaultAttributes sets attributes to each span.
type DefaultAttributes []trace.Attribute

func (d DefaultAttributes) apply(c *callbacks) {
	c.defaultAttributes = []trace.Attribute(d)
}

type callbacks struct {
	// Allow ocgorm to create root spans absence of existing spans or even context.
	// Default is to not trace ocgorm calls if no existing parent span is found
	// in context.
	allowRoot bool

	// Allow recording of sql queries in spans.
	// Only allow this if it is safe to have queries recorded with respect to
	// security.
	query bool

	// startOptions are applied to the span started around each request.
	//
	// StartOptions.SpanKind will always be set to trace.SpanKindClient.
	startOptions trace.StartOptions

	// DefaultAttributes will be set to each span as default.
	defaultAttributes []trace.Attribute
}

// RegisterCallbacks registers the necessary callbacks in Gorm's hook system for instrumentation.
func RegisterCallbacks(db *gorm.DB, opts ...Option) {
	c := &callbacks{
		defaultAttributes: []trace.Attribute{},
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	db.Callback().Create().Before("gorm:create").Register("instrumentation:before_create", c.beforeCreate)
	db.Callback().Create().After("gorm:create").Register("instrumentation:after_create", c.afterCreate)
	db.Callback().Query().Before("gorm:query").Register("instrumentation:before_query", c.beforeQuery)
	db.Callback().Query().After("gorm:query").Register("instrumentation:after_query", c.afterQuery)
	db.Callback().Update().Before("gorm:update").Register("instrumentation:before_update", c.beforeUpdate)
	db.Callback().Update().After("gorm:update").Register("instrumentation:after_update", c.afterUpdate)
	db.Callback().Delete().Before("gorm:delete").Register("instrumentation:before_delete", c.beforeDelete)
	db.Callback().Delete().After("gorm:delete").Register("instrumentation:after_delete", c.afterDelete)
}

func (c *callbacks) before(scope *gorm.Scope, operation string) {
	rctx, _ := scope.Get(contextScopeKey)
	ctx, ok := rctx.(context.Context)
	if !ok || ctx == nil {
		ctx = context.Background()
	}

	ctx = c.startTrace(ctx, scope, operation)
	ctx = c.startStats(ctx, scope, operation)

	scope.Set(contextScopeKey, ctx)
}

func (c *callbacks) after(scope *gorm.Scope) {
	c.endTrace(scope)
	c.endStats(scope)
}

func (c *callbacks) startTrace(ctx context.Context, scope *gorm.Scope, operation string) context.Context {
	// Context is missing, but we allow root spans to be created
	if ctx == nil {
		ctx = context.Background()
	}

	parentSpan := trace.FromContext(ctx)
	if parentSpan == nil && !c.allowRoot {
		return ctx
	}

	var span *trace.Span

	if parentSpan == nil {
		ctx, span = trace.StartSpan(
			context.Background(),
			fmt.Sprintf("gorm:%s", operation),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithSampler(c.startOptions.Sampler),
		)
	} else {
		_, span = trace.StartSpan(ctx, fmt.Sprintf("gorm:%s", operation))
	}

	attributes := append(
		c.defaultAttributes,
		trace.StringAttribute(TableAttribute, scope.TableName()),
	)

	if c.query {
		attributes = append(attributes, trace.StringAttribute(QueryAttribute, scope.SQL))
	}

	span.AddAttributes(attributes...)

	scope.Set(spanScopeKey, span)

	return ctx
}

func (c *callbacks) endTrace(scope *gorm.Scope) {
	rspan, ok := scope.Get(spanScopeKey)
	if !ok {
		return
	}

	span, ok := rspan.(*trace.Span)
	if !ok {
		return
	}

	var status trace.Status

	if scope.HasError() {
		err := scope.DB().Error
		if gorm.IsRecordNotFoundError(err) {
			status.Code = trace.StatusCodeNotFound
		} else {
			status.Code = trace.StatusCodeUnknown
		}

		status.Message = err.Error()
	}

	span.SetStatus(status)

	span.End()
}

func (c *callbacks) startStats(ctx context.Context, scope *gorm.Scope, operation string) context.Context {
	ctx, _ = tag.New(ctx, tag.Upsert(Operation, operation), tag.Upsert(Table, scope.TableName()))

	return ctx
}

func (c *callbacks) endStats(scope *gorm.Scope) {
	if scope.HasError() {
		return
	}

	rctx, _ := scope.Get(contextScopeKey)
	ctx, ok := rctx.(context.Context)
	if !ok || ctx == nil {
		return
	}

	stats.Record(ctx, QueryCount.M(1))
}

func (c *callbacks) beforeCreate(scope *gorm.Scope) { c.before(scope, "create") }
func (c *callbacks) afterCreate(scope *gorm.Scope)  { c.after(scope) }
func (c *callbacks) beforeQuery(scope *gorm.Scope)  { c.before(scope, "query") }
func (c *callbacks) afterQuery(scope *gorm.Scope)   { c.after(scope) }
func (c *callbacks) beforeUpdate(scope *gorm.Scope) { c.before(scope, "update") }
func (c *callbacks) afterUpdate(scope *gorm.Scope)  { c.after(scope) }
func (c *callbacks) beforeDelete(scope *gorm.Scope) { c.before(scope, "delete") }
func (c *callbacks) afterDelete(scope *gorm.Scope)  { c.after(scope) }
