package main

import (
	"fmt"
	"net/http"
	"os"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // blank import is used here for simplicity
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	"github.com/hashicorp/go-gin-gorm-opencensus/internal"
	"github.com/hashicorp/go-gin-gorm-opencensus/pkg/ocgorm"
)

func main() {
	// Create prometheus exporter
	pe, err := prometheus.NewExporter(prometheus.Options{
		Registry: prom.DefaultGatherer.(*prom.Registry),
	})
	if err != nil {
		panic(err)
	}

	// Register prometheus as a stats exporter
	view.RegisterExporter(pe)

	// Register stat views
	err = view.Register(
		// Gin (HTTP) stats
		ochttp.ServerRequestCountView,
		ochttp.ServerRequestBytesView,
		ochttp.ServerResponseBytesView,
		ochttp.ServerLatencyView,
		ochttp.ServerRequestCountByMethod,
		ochttp.ServerResponseCountByStatusCode,

		// Gorm stats
		ocgorm.SQLClientCallsView,
	)
	if err != nil {
		panic(err)
	}

	// Always trace for this demo. In a production application, you should
	// configure this to a trace.ProbabilitySampler set at the desired
	// probability.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	// Create jaeger exporter
	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint:     os.Getenv("JAEGER_AGENT_ENDPOINT"),
		CollectorEndpoint: os.Getenv("JAEGER_ENDPOINT"),
		Process: jaeger.Process{
			ServiceName: "go-gin-gorm-opencensus",
		},
	})
	if err != nil {
		panic(err)
	}

	// Register jaeger as a trace exporter
	trace.RegisterExporter(je)

	// Connect to database
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	// Register instrumentation callbacks
	ocgorm.RegisterCallbacks(db)

	// Run migrations and fixtures
	db.AutoMigrate(internal.Person{})
	err = internal.Fixtures(db)
	if err != nil {
		panic(err)
	}

	// Initialize Gin engine
	r := gin.Default()

	r.GET("/metrics", gin.HandlerFunc(func(c *gin.Context) {
		pe.ServeHTTP(c.Writer, c.Request)
	}))

	// Add routes
	r.POST(
		"/people",
		func(c *gin.Context) {
			ochttp.SetRoute(c.Request.Context(), "/people")
		},
		internal.CreatePerson(db),
	)
	r.GET(
		"/hello/:firstName",
		func(c *gin.Context) {
			ochttp.SetRoute(c.Request.Context(), "/hello/:firstName")
		},
		internal.Hello(db),
	)

	// Listen and serve on 0.0.0.0:8080
	address := "127.0.0.1:8080"
	fmt.Printf("Listening and serving HTTP on %s\n", address)
	http.ListenAndServe( // nolint: errcheck
		address,
		&ochttp.Handler{
			Handler: r,
			GetStartOptions: func(r *http.Request) trace.StartOptions {
				startOptions := trace.StartOptions{}

				if r.URL.Path == "/metrics" {
					startOptions.Sampler = trace.NeverSample()
				}

				return startOptions
			},
		},
	)
}
