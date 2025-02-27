// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package storage provides the functions that interacts with Spanner to fetch the data
package storage

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/cloudspannerecosystem/dynamodb-adapter/models"
	otelgo "github.com/cloudspannerecosystem/dynamodb-adapter/otel"
	"github.com/cloudspannerecosystem/dynamodb-adapter/pkg/logger"
	"go.opentelemetry.io/otel"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

// Storage object for intracting with storage package
type Storage struct {
	spannerClient map[string]*spanner.Client
}

// storage - global instance of storage
var storage *Storage

func InitializeDriver(ctx context.Context) error {
	if models.GlobalConfig == nil {
		return fmt.Errorf("GlobalConfig is not initialized")
	}
	storage = &Storage{
		spannerClient: make(map[string]*spanner.Client),
	}

	// OpenTelemetry configuration
	otelConfig := otelgo.OTelConfig{
		TracerEndpoint:     models.GlobalConfig.Otel.Traces.Endpoint,
		MetricEndpoint:     models.GlobalConfig.Otel.Metrics.Endpoint,
		ServiceName:        models.GlobalConfig.Otel.ServiceName,
		MetricsEnabled:     models.GlobalConfig.Otel.Metrics.Enabled,
		TracesEnabled:      models.GlobalConfig.Otel.Traces.Enabled,
		TraceSampleRatio:   models.GlobalConfig.Otel.Traces.SamplingRatio,
		Database:           models.GlobalConfig.Spanner.DatabaseName,
		Instance:           models.GlobalConfig.Spanner.InstanceID,
		HealthCheckEnabled: models.GlobalConfig.Otel.HealthCheck.Enabled,
		HealthCheckEp:      models.GlobalConfig.Otel.HealthCheck.Endpoint,
	}

	otelInstance, shutdownOTel, err := otelgo.NewOpenTelemetry(ctx, &otelConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}
	defer func() {
		if err != nil && shutdownOTel != nil {
			_ = shutdownOTel(ctx)
		}
	}()

	spc := spanner.DefaultSessionPoolConfig

	if models.GlobalConfig.Spanner.Session.Min != 0 {
		spc.MinOpened = models.GlobalConfig.Spanner.Session.Min
	}

	if models.GlobalConfig.Spanner.Session.Max != 0 {
		spc.MaxOpened = models.GlobalConfig.Spanner.Session.Max
	}

	spc.InactiveTransactionRemovalOptions = spanner.InactiveTransactionRemovalOptions{
		ActionOnInactiveTransaction: spanner.WarnAndClose,
	}

	cfg := spanner.ClientConfig{SessionPoolConfig: spc, UserAgent: models.GlobalConfig.UserAgent}

	// If OpenTelemetry is provided, configure instrumentation
	if models.GlobalConfig.Otel.Enabled && otelInstance != nil {
		if models.GlobalConfig.Otel.EnabledClientSideMetrics {
			// Enable OpenTelemetry metrics before injecting meter provider.
			spanner.EnableOpenTelemetryMetrics()
		}
		// Setting the GOOGLE_API_GO_EXPERIMENTAL_TELEMETRY_PLATFORM_TRACING env varibale to 'opentelemetry' will enable traces on spanner client library.
		os.Setenv("GOOGLE_API_GO_EXPERIMENTAL_TELEMETRY_PLATFORM_TRACING", "opentelemetry")

		// Set up OpenTelemetry traces and metrics
		otel.SetTracerProvider(otelInstance.TracerProvider)

		// Add OpenTelemetry instrumentation to Spanner client configuration
		cfg.OpenTelemetryMeterProvider = otelInstance.MeterProvider
	}
	// Spanner client initialization
	database := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		models.GlobalConfig.Spanner.ProjectID,
		models.GlobalConfig.Spanner.InstanceID,
		models.GlobalConfig.Spanner.DatabaseName,
	)
	spannerClient, err := spanner.NewClientWithConfig(ctx, database, cfg,
		option.WithGRPCConnectionPool(models.GlobalConfig.Spanner.Session.GrpcChannels),
		option.WithGRPCDialOption(grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 10 * time.Second,
		})),
	)
	if err != nil {
		return fmt.Errorf("failed to create Spanner client: %w", err)
	}

	storage.spannerClient[models.GlobalConfig.Spanner.InstanceID] = spannerClient
	logger.LogInfo("Spanner client initialized successfully")

	if models.GlobalConfig.Otel.Traces.Enabled {
		if models.GlobalConfig.Otel.Traces.SamplingRatio < 0 || models.GlobalConfig.Otel.Traces.SamplingRatio > 1 {
			return fmt.Errorf("sampling ratio for Otel Traces should be between 0 and 1]")
		}
	}

	models.GlobalProxy = &models.Proxy{}
	models.GlobalProxy.OtelInst = otelInstance
	models.GlobalProxy.OtelShutdown = shutdownOTel
	return nil
}

// Close - This gracefully returns the session pool objects, when driver gets exit signal
func (s Storage) Close() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	logger.LogDebug("Connection Shutdown start")
	for _, v := range s.spannerClient {
		v.Close()
	}
	logger.LogDebug("Connection shutted down")
}

var once sync.Once

// GetStorageInstance - return storage instance to call db functionalities
func GetStorageInstance() *Storage {
	once.Do(func() {
		if storage == nil {
			_ = InitializeDriver(context.Background())
		}
	})
	return storage
}

func (s Storage) getSpannerClient(_ string) *spanner.Client {
	return s.spannerClient[models.GlobalConfig.Spanner.InstanceID]
}
