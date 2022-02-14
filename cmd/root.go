package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/numary/go-libs-cloud/pkg/middlewares"
	"github.com/numary/payment/pkg"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/rs/cors"
	"github.com/spf13/cobra"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	_ "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	_ "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	_ "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

const (
	mongodbUriFlag                       = "mongodb-uri"
	mongodbDatabaseFlag                  = "mongodb-database"
	authUriFlag                          = "auth-uri"
	otelTracesFlag                       = "otel-traces"
	otelTracesExporterFlag               = "otel-traces-exporter"
	otelTracesExporterJaegerEndpointFlag = "otel-traces-exporter-jaeger-endpoint"
	otelTracesExporterJaegerUserFlag     = "otel-traces-exporter-jaeger-user"
	otelTracesExporterJaegerPasswordFlag = "otel-traces-exporter-jaeger-password"
	otelTracesExporterOTLPModeFlag       = "otel-traces-exporter-otlp-mode"
	otelTracesExporterOTLPEndpointFlag   = "otel-traces-exporter-otlp-endpoint"
	otelTracesExporterOTLPInsecureFlag   = "otel-traces-exporter-otlp-insecure"
)

var rootCmd = &cobra.Command{
	Use:   "payment",
	Short: "Payment api",
	RunE: func(cmd *cobra.Command, args []string) error {

		mongodbUri := viper.GetString(mongodbUriFlag)
		if mongodbUri == "" {
			return errors.New("missing mongodb uri")
		}

		mongodbDatabase := viper.GetString(mongodbDatabaseFlag)
		if mongodbDatabase == "" {
			return errors.New("missing mongodb database name")
		}

		authUri := viper.GetString(authUriFlag)
		if authUri == "" {
			return errors.New("missing auth uri")
		}

		client, err := mongo.NewClient(options.Client().ApplyURI(mongodbUri))
		if err != nil {
			return err
		}

		logrus.Infoln("Connection on database: " + mongodbUri)
		err = client.Connect(context.Background())
		if err != nil {
			return err
		}

		logrus.Infoln("Ping database...")
		ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			return err
		}

		db := client.Database(mongodbDatabase)

		if viper.GetBool(otelTracesFlag) {
			var exporter sdktrace.SpanExporter
			switch viper.GetString(otelTracesExporterFlag) {
			case "stdout":
				exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
			case "jaeger":
				options := make([]jaeger.CollectorEndpointOption, 0)
				if ep := viper.GetString(otelTracesExporterJaegerEndpointFlag); ep != "" {
					options = append(options, jaeger.WithEndpoint(ep))
				}
				if username := viper.GetString(otelTracesExporterJaegerUserFlag); username != "" {
					options = append(options, jaeger.WithUsername(username))
				}
				if password := viper.GetString(otelTracesExporterJaegerPasswordFlag); password != "" {
					options = append(options, jaeger.WithPassword(password))
				}
				exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(options...))
			case "otlp":
				var client otlptrace.Client
				switch viper.GetString(otelTracesExporterOTLPModeFlag) {
				case "http":
					options := make([]otlptracehttp.Option, 0)
					if insecure := viper.GetBool(otelTracesExporterOTLPInsecureFlag); insecure {
						options = append(options, otlptracehttp.WithInsecure())
					}
					if endpoint := viper.GetString(otelTracesExporterOTLPEndpointFlag); endpoint != "" {
						options = append(options, otlptracehttp.WithEndpoint(endpoint))
					}
					client = otlptracehttp.NewClient(options...)
				case "grpc":
					options := make([]otlptracegrpc.Option, 0)
					if insecure := viper.GetBool(otelTracesExporterOTLPInsecureFlag); insecure {
						options = append(options, otlptracegrpc.WithInsecure())
					}
					if endpoint := viper.GetString(otelTracesExporterOTLPEndpointFlag); endpoint != "" {
						options = append(options, otlptracegrpc.WithEndpoint(endpoint))
					}
					client = otlptracegrpc.NewClient()
				}
				exporter, err = otlptrace.New(context.Background(), client)
			}
			if err != nil {
				return err
			}

			tp := sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String("Payments"))),
			)
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
			defer tp.Shutdown(context.Background())
		}

		s := payment.NewDefaultService(db)

		mux := payment.NewMux(s)
		if viper.GetBool(otelTracesFlag) {
			mux.Use(otelmux.Middleware("Payments"))
		}
		mux.Use(
			payment.Recovery(func(ctx context.Context, e interface{}) {
				if viper.GetBool(otelTracesFlag) {
					switch err := e.(type) {
					case error:
						trace.SpanFromContext(ctx).RecordError(err, trace.WithStackTrace(true))
						trace.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
					default:
						trace.SpanFromContext(ctx).RecordError(fmt.Errorf("%s", e), trace.WithStackTrace(true))
						trace.SpanFromContext(ctx).SetStatus(codes.Error, fmt.Sprintf("%s", e))
					}
				} else {
					logrus.Errorln(e)
					debug.PrintStack()
				}
			}),
			cors.New(cors.Options{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut},
				AllowCredentials: true,
			}).Handler,
			middlewares.AuthMiddleware(authUri),
			payment.CheckOrganizationAccessMiddleware(),
		)
		logrus.Infoln("Listening on port 8080...")
		return http.ListenAndServe(":8080", mux)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().String(mongodbUriFlag, "mongodb://localhost:27017", "MongoDB address")
	rootCmd.Flags().String(mongodbDatabaseFlag, "payments", "MongoDB database name")
	rootCmd.Flags().String(authUriFlag, "auth-uri", "Auth uri")
	rootCmd.Flags().Bool(otelTracesFlag, false, "Enable OpenTelemetry traces support")
	rootCmd.Flags().String(otelTracesExporterFlag, "stdout", "OpenTelemetry traces exporter")
	rootCmd.Flags().String(otelTracesExporterJaegerEndpointFlag, "", "OpenTelemetry traces Jaeger exporter endpoint")
	rootCmd.Flags().String(otelTracesExporterJaegerUserFlag, "", "OpenTelemetry traces Jaeger exporter user")
	rootCmd.Flags().String(otelTracesExporterJaegerPasswordFlag, "", "OpenTelemetry traces Jaeger exporter password")
	rootCmd.Flags().String(otelTracesExporterOTLPModeFlag, "grpc", "OpenTelemetry traces OTLP exporter mode (grpc|http)")
	rootCmd.Flags().String(otelTracesExporterOTLPEndpointFlag, "", "OpenTelemetry traces grpc endpoint")
	rootCmd.Flags().Bool(otelTracesExporterOTLPInsecureFlag, false, "OpenTelemetry traces grpc insecure")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(rootCmd.Flags())
}
