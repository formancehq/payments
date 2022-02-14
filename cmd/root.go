package cmd

import (
	"errors"
	"github.com/numary/go-libs-cloud/pkg/middlewares"
	"github.com/numary/payment/pkg"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rs/cors"
	"github.com/spf13/cobra"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	mongodbUriFlag      = "mongodb-uri"
	mongodbDatabaseFlag = "mongodb-database"
	authUriFlag         = "auth-uri"
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

		db := client.Database(mongodbDatabase)

		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			log.Fatal(err)
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String("Payments"))),
		)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

		s := payment.NewDefaultService(db)
		var handler http.Handler
		handler = payment.ConfigureAuthMiddleware(
			payment.NewMux(s),
			middlewares.AuthMiddleware(authUri),
			payment.CheckOrganizationAccessMiddleware(),
		)
		handler = payment.Recovery(handler)
		handler = otelhttp.NewHandler(handler, "Payments")
		handler = cors.New(cors.Options{
			AllowedOrigins: []string{"*"},

			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut},
			AllowCredentials: true,
		}).Handler(handler)

		return http.ListenAndServe(":8080", handler)
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

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(rootCmd.Flags())
}
