package ocgin

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

// Handler is a gin.HandlerFunc wrapper to instrument your Gin HTTP server with
// OpenCensus. It supports both stats and tracing.
//
// Tracing
//
// This handler is aware of the incoming request's span, reading it from request
// headers as configured using the Propagation field.
// The extracted span can be accessed from the incoming request's
// context.
//
//    span := trace.FromContext(c.Request.Context())
//
// The server span will be automatically ended at the end of ServeHTTP.
//
// The implementation is heavily based on https://godoc.org/go.opencensus.io/plugin/ochttp
type Handler struct {
	// Propagation defines how traces are propagated. If unspecified,
	// B3 propagation will be used.
	Propagation propagation.HTTPFormat

	// StartOptions are applied to the span started by this Handler around each
	// request.
	//
	// StartOptions.SpanKind will always be set to trace.SpanKindServer
	// for spans started by this transport.
	StartOptions trace.StartOptions

	// GetStartOptions allows to set start options per request. If set,
	// StartOptions is going to be ignored.
	GetStartOptions func(ctx *gin.Context) trace.StartOptions

	// IsPublicEndpoint should be set to true for publicly accessible HTTP(S)
	// servers. If true, any trace metadata set on the incoming request will
	// be added as a linked trace instead of being added as a parent of the
	// current trace.
	IsPublicEndpoint bool

	// FormatSpanName holds the function to use for generating the span name
	// from the information found in the incoming HTTP Request. By default the
	// name equals the URL Path.
	FormatSpanName func(ctx *gin.Context) string
}

func (h *Handler) HandlerFunc(c *gin.Context) {
	var tags addedTags
	traceEnd := h.startTrace(c)
	defer traceEnd()
	statsEnd := h.startStats(c)
	defer statsEnd(&tags)

	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), addedTagsKey{}, &tags))

	c.Next()
}

func (h *Handler) startTrace(c *gin.Context) func() {
	if isHealthOrMetricsEndpoint(c.Request.URL.Path) {
		return func() {}
	}
	var name string
	if h.FormatSpanName == nil {
		name = spanNameFromURL(c)
	} else {
		name = h.FormatSpanName(c)
	}
	ctx := c.Request.Context()

	startOpts := h.StartOptions
	if h.GetStartOptions != nil {
		startOpts = h.GetStartOptions(c)
	}

	var span *trace.Span
	sc, ok := h.extractSpanContext(c.Request)
	if ok && !h.IsPublicEndpoint {
		ctx, span = trace.StartSpanWithRemoteParent(ctx, name, sc,
			trace.WithSampler(startOpts.Sampler),
			trace.WithSpanKind(trace.SpanKindServer))
	} else {
		ctx, span = trace.StartSpan(ctx, name,
			trace.WithSampler(startOpts.Sampler),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		if ok {
			span.AddLink(trace.Link{
				TraceID:    sc.TraceID,
				SpanID:     sc.SpanID,
				Type:       trace.LinkTypeChild,
				Attributes: nil,
			})
		}
	}
	span.AddAttributes(requestAttrs(c.Request)...)
	c.Request = c.Request.WithContext(ctx)
	return span.End
}

func (h *Handler) extractSpanContext(r *http.Request) (trace.SpanContext, bool) {
	if h.Propagation == nil {
		return defaultFormat.SpanContextFromRequest(r)
	}
	return h.Propagation.SpanContextFromRequest(r)
}

func (h *Handler) startStats(c *gin.Context) func(tags *addedTags) {
	ctx, _ := tag.New(c.Request.Context(),
		tag.Upsert(ochttp.Host, c.Request.URL.Host),
		tag.Upsert(ochttp.Path, c.Request.URL.Path),
		tag.Upsert(ochttp.Method, c.Request.Method),
		tag.Insert(ochttp.KeyServerRoute, c.HandlerName()))
	track := &trackingResponseWriter{
		start:          time.Now(),
		ctx:            ctx,
		ResponseWriter: c.Writer,
	}
	if c.Request.Body == nil {
		// TODO: Handle cases where ContentLength is not set.
		track.reqSize = -1
	} else if c.Request.ContentLength > 0 {
		track.reqSize = c.Request.ContentLength
	}
	stats.Record(ctx, ochttp.ServerRequestCount.M(1))

	c.Writer = track

	return track.end
}

type trackingResponseWriter struct {
	gin.ResponseWriter

	ctx     context.Context
	reqSize int64
	start   time.Time
	endOnce sync.Once
}

// Compile time assertion for ResponseWriter interface
var _ gin.ResponseWriter = (*trackingResponseWriter)(nil)

func (t *trackingResponseWriter) end(tags *addedTags) {
	t.endOnce.Do(func() {
		status := t.Status()
		if status == 0 {
			status = 200
		}

		span := trace.FromContext(t.ctx)
		span.SetStatus(ochttp.TraceStatus(status, http.StatusText(status)))
		span.AddAttributes(trace.Int64Attribute(ochttp.StatusCodeAttribute, int64(status)))

		m := []stats.Measurement{
			ochttp.ServerLatency.M(float64(time.Since(t.start)) / float64(time.Millisecond)),
			ochttp.ServerResponseBytes.M(int64(t.Size())),
		}
		if t.reqSize >= 0 {
			m = append(m, ochttp.ServerRequestBytes.M(t.reqSize))
		}
		allTags := make([]tag.Mutator, len(tags.t)+1)
		allTags[0] = tag.Upsert(ochttp.StatusCode, strconv.Itoa(status))
		copy(allTags[1:], tags.t)
		stats.RecordWithTags(t.ctx, allTags, m...) // nolint: errcheck
	})
}
