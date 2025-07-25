package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/viper"

	"github.com/example/wpp-wave-bot/internal/api"
	"github.com/example/wpp-wave-bot/internal/db"
	"github.com/example/wpp-wave-bot/internal/db/seeders"
	"github.com/example/wpp-wave-bot/internal/rabbitmq"
	"github.com/example/wpp-wave-bot/internal/whatsapp"
)

func main() {
	// Setup logger with human friendly console output
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Load configuration from config.yaml and environment variables
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("unable to read config file, relying on env vars")
	}

	// Parse command, default to run service
	cmd := "run"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize database connection pool
	dbPool, err := db.New(viper.GetString("database_url"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect database")
	}
	defer dbPool.Close()

	switch cmd {
	case "migrate":
		if err := db.Migrate(viper.GetString("database_url"), "internal/db/migrations"); err != nil {
			log.Fatal().Err(err).Msg("failed to run migrations")
		}
		return
	case "seed":
		if err := db.Migrate(viper.GetString("database_url"), "internal/db/migrations"); err != nil {
			log.Fatal().Err(err).Msg("failed to run migrations")
		}
		if err := seeders.Seed(dbPool); err != nil {
			log.Fatal().Err(err).Msg("failed to seed database")
		}
		log.Info().Msg("seeding completed")
		return
	case "run":
		if err := db.Migrate(viper.GetString("database_url"), "internal/db/migrations"); err != nil {
			log.Fatal().Err(err).Msg("failed to run migrations")
		}
		// continue below
	default:
		log.Fatal().Msgf("unknown command %s", cmd)
	}

	// Initialize RabbitMQ
	mq, err := rabbitmq.New(viper.GetString("rabbitmq_url"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to rabbitmq")
	}
	defer mq.Close()

	// Initialize WhatsApp handler
	wa, err := whatsapp.New(dbPool, viper.GetString("database_url"), mq)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init whatsapp client")
	}

	// start admin API server
	apiSrv := api.New(wa)
	go func() {
		addr := viper.GetString("http_addr")
		if addr == "" {
			addr = ":8080"
		}
		if err := apiSrv.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("api server exited")
			stop()
		}
	}()

	if err := wa.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("whatsapp service exited")
	}
}
