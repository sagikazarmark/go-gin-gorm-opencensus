package ocgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

var defaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}

func isHealthOrMetricsEndpoint(path string) bool {
	// Health checking is pretty frequent and
	// traces collected for health endpoints
	// can be extremely noisy and expensive.
	// Disable canonical health checking endpoints
	// like /healthz and /_ah/health for now.
	if path == "/healthz" || path == "/_ah/health" || path == "/metrics" {
		return true
	}
	return false
}

func spanNameFromURL(c *gin.Context) string {
	return c.Request.URL.Path
}

func requestAttrs(r *http.Request) []trace.Attribute {
	return []trace.Attribute{
		trace.StringAttribute(ochttp.PathAttribute, r.URL.Path),
		trace.StringAttribute(ochttp.HostAttribute, r.URL.Host),
		trace.StringAttribute(ochttp.MethodAttribute, r.Method),
		trace.StringAttribute(ochttp.UserAgentAttribute, r.UserAgent()),
	}
}
