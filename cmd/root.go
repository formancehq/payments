package cmd

import (
	"errors"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"os"
	payment "payment/pkg"
	"strings"

	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

const (
	mongodbUriFlag      = "mongodb-uri"
	mongodbDatabaseFlag = "mongodb-database"
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

		client, err := mongo.NewClient(options.Client().ApplyURI(mongodbUri))
		if err != nil {
			return err
		}

		db := client.Database(mongodbDatabase)

		s := payment.NewDefaultService(db)
		var handler http.Handler
		handler = payment.NewMux(s)
		handler = cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
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

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(rootCmd.Flags())
}
