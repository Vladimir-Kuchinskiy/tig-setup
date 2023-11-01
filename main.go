package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	runGracefulShutDownListener := func() {
		osCall := <-c
		logger.Info("stop syscall", "code", osCall.String())
		cancel()
	}

	go runGracefulShutDownListener()

	recordsCollectionName := "records"

	mongodbClient, err := connectToMongoDB()
	if err != nil {
		panic(err)
	}

	elasticsearchClient, err := connectToElasticsearch()
	if err != nil {
		panic(err)
	}
	if _, err := elasticsearchClient.Indices.Create(recordsCollectionName); err != nil {
		panic(err)
	}

	var counter atomic.Uint64

	handleMetrics := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Write([]byte("TODO: metrics"))
	}
	handleHealthz := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Write([]byte("healthy"))
	}
	handleLoad := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		counter.Add(1)

		count := counter.Load()

		record := Record{
			ID:   count,
			Name: fmt.Sprintf("Record-%d", count),
		}

		errorGroup, ctx := errgroup.WithContext(ctx)

		errorGroup.Go(func() error {
			_, err := mongodbClient.Database("test-db").Collection(recordsCollectionName).
				InsertOne(ctx, bson.D{{"id", record.ID}, {"name", record.Name}})
			if err != nil {
				return err
			}

			return nil
		})

		errorGroup.Go(func() error {
			rawRecord, err := json.Marshal(record)
			if err != nil {
				return err
			}

			if _, err = elasticsearchClient.Index(recordsCollectionName, bytes.NewReader(rawRecord)); err != nil {
				return err
			}

			return nil
		})

		if err := errorGroup.Wait(); err != nil {
			logger.Error("failed to perform operation", "error", err)

			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("error"))
			if err != nil {
				logger.Error("failed to write to response object", "error", err)
			}
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Error("failed to perform operation", "error", err)
			return
		}
	}

	router := httprouter.New()

	router.GET("/metrics", handleMetrics)
	router.GET("/healthz", handleHealthz)
	router.GET("/load", handleLoad)

	server := http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           router,
		ReadHeaderTimeout: time.Minute,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			cancel()
		}
	}()

	logger.Info("Server started listening", "address", server.Addr)

	<-ctx.Done()
	logger.Info("Received shutdown signal ...")

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("failed to gracefully close the server", "error", err)
		os.Exit(1)
	}

	logger.Info("server gracefully closed")
}

type Record struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

func connectToMongoDB() (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := os.Getenv("MONGODB_HOST")
	if host == "" {
		host = "localhost"
	}
	username := os.Getenv("MONGODB_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("MONGODB_PASSWORD")
	if password == "" {
		password = "admin"
	}

	connectionURL := fmt.Sprintf("mongodb://%s:%s@%s:27017", username, password, host)

	return mongo.Connect(ctx, options.Client().ApplyURI(connectionURL))
}

func connectToElasticsearch() (*elasticsearch.Client, error) {
	host := os.Getenv("ELASTICSEARCH_HOST")
	if host == "" {
		host = "localhost"
	}

	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{fmt.Sprintf("http://%s:9200", host)},
	})
}
