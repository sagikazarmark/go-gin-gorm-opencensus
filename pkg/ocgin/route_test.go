package ocgin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func TestWithRouteTag(t *testing.T) {
	v := &view.View{
		Name:        "request_total",
		Measure:     ochttp.ServerLatency,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ochttp.KeyServerRoute},
	}
	view.Register(v)
	var e testStatsExporter
	view.RegisterExporter(&e)
	defer view.UnregisterExporter(&e)

	en := gin.New()
	en.Use((&Handler{}).HandlerFunc)
	en.Any(
		"/a/b/c",
		WithRouteTag(gin.HandlerFunc(func(c *gin.Context) {
			c.Writer.WriteHeader(204)
		}), "/a/"),
	)

	req, _ := http.NewRequest("GET", "/a/b/c", nil)
	rr := httptest.NewRecorder()
	en.ServeHTTP(rr, req)
	if got, want := rr.Code, 204; got != want {
		t.Fatalf("Unexpected response, got %d; want %d", got, want)
	}

	view.Unregister(v) // trigger exporting

	got := e.rowsForView("request_total")
	want := []*view.Row{
		{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{Key: ochttp.KeyServerRoute, Value: "/a/"}}},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Unexpected view data exported, -got, +want: %s", diff)
	}
}

func TestTracedRouter(t *testing.T) {
	v := &view.View{
		Name:        "request_total",
		Measure:     ochttp.ServerLatency,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ochttp.KeyServerRoute},
	}
	view.Register(v)
	var e testStatsExporter
	view.RegisterExporter(&e)
	defer view.UnregisterExporter(&e)

	en := gin.New()
	en.Use((&Handler{}).HandlerFunc)
	router := NewTracedRouter(en)
	router.Any(
		"/a/:b/:c",
		gin.HandlerFunc(func(c *gin.Context) {
			c.Writer.WriteHeader(204)
		}),
	)

	req, _ := http.NewRequest("GET", "/a/b/c", nil)
	rr := httptest.NewRecorder()
	en.ServeHTTP(rr, req)
	if got, want := rr.Code, 204; got != want {
		t.Fatalf("Unexpected response, got %d; want %d", got, want)
	}

	view.Unregister(v) // trigger exporting

	got := e.rowsForView("request_total")
	want := []*view.Row{
		{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{Key: ochttp.KeyServerRoute, Value: "/a/:b/:c"}}},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Unexpected view data exported, -got, +want: %s", diff)
	}
}

type testStatsExporter struct {
	vd []*view.Data
}

func (t *testStatsExporter) ExportView(d *view.Data) {
	t.vd = append(t.vd, d)
}

func (t *testStatsExporter) rowsForView(name string) []*view.Row {
	var rows []*view.Row
	for _, d := range t.vd {
		if d.View.Name == name {
			rows = append(rows, d.Rows...)
		}
	}
	return rows
}
