package ocgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/tag"
)

type RouterGroup interface {
	IRouter

	BasePath() string
}

type IRouter interface {
	gin.IRoutes

	Group(string, ...gin.HandlerFunc) RouterGroup
}

// NewTracedRouter returns a router which adds the route to trace information.
func NewTracedRouter(engine *gin.Engine) RouterGroup {
	return &tracedRouterGroup{
		router: &engine.RouterGroup,
	}
}

type tracedRouterGroup struct {
	router *gin.RouterGroup
}

func (g *tracedRouterGroup) Use(middleware ...gin.HandlerFunc) gin.IRoutes {
	g.router.Use(middleware...)

	return g
}

func (g *tracedRouterGroup) Group(relativePath string, handlers ...gin.HandlerFunc) RouterGroup {
	return &tracedRouterGroup{
		router: g.router.Group(relativePath, handlers...),
	}
}

func (g *tracedRouterGroup) BasePath() string {
	return g.router.BasePath()
}

func (g *tracedRouterGroup) Handle(httpMethod string, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.Handle(httpMethod, relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) Any(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.Any(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.GET(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.POST(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.DELETE(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) PATCH(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.PATCH(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.PUT(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.OPTIONS(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) HEAD(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	g.router.HEAD(relativePath, AllWithRouteTag(relativePath, handlers...)...)

	return g
}

func (g *tracedRouterGroup) StaticFile(relativePath string, filePath string) gin.IRoutes {
	g.router.StaticFile(relativePath, filePath)

	return g
}

func (g *tracedRouterGroup) Static(relativePath string, root string) gin.IRoutes {
	g.router.Static(relativePath, root)

	return g
}

func (g *tracedRouterGroup) StaticFS(relativePath string, fs http.FileSystem) gin.IRoutes {
	g.router.StaticFS(relativePath, fs)

	return g
}

// WithRouteTag returns a gin.HandlerFunc that records stats with the
// http_server_route tag set to the given value.
func WithRouteTag(handler gin.HandlerFunc, route string) gin.HandlerFunc {
	return taggedHandlerFunc(func(c *gin.Context) []tag.Mutator {
		addRoute := []tag.Mutator{tag.Upsert(ochttp.KeyServerRoute, route)}
		ctx, _ := tag.New(c.Request.Context(), addRoute...)
		c.Request = c.Request.WithContext(ctx)
		handler(c)
		return addRoute
	}).HandlerFunc
}

// AllWithRouteTag does the same as WithRouteTag for a whole handler chain.
func AllWithRouteTag(route string, handlers ...gin.HandlerFunc) []gin.HandlerFunc {
	for key, value := range handlers {
		handlers[key] = WithRouteTag(value, route)
	}

	return handlers
}

// taggedHandlerFunc is a gin.HandlerFunc that returns tags describing the
// processing of the request. These tags will be recorded along with the
// measures in this package at the end of the request.
type taggedHandlerFunc func(c *gin.Context) []tag.Mutator

func (h taggedHandlerFunc) HandlerFunc(c *gin.Context) {
	tags := h(c)
	if a, ok := c.Request.Context().Value(addedTagsKey{}).(*addedTags); ok {
		a.t = append(a.t, tags...)
	}
}
