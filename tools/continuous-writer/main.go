package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Simple continuous writer that inserts rows into payments.payment table.
// It is external to the main app and can be run ad-hoc during migrations.
//
// Env vars (overridden by flags):
//   PG_DSN: Postgres DSN (e.g. postgres://user:pass@localhost:5432/payments?sslmode=disable)
//   INTERVAL_MS: delay between inserts per worker
//   WORKERS: number of concurrent goroutines inserting
//   CONNECTOR_ID: connector id to stamp on rows
//   ASSET: asset string (e.g. USD/2)
//   SCHEME: payment scheme string (e.g. card, bank_transfer)
//   TYPE: payin|payout|transfer
//   STATUS: pending|succeeded|failed|cancelled

type paymentRow struct {
	id         string
	ref        string
	initAmount int
	amount     int
	source     any
	dest       any
}

func main() {
	var (
		dsn               = getEnv("PG_DSN", "postgres://payments:payments@localhost:5432/payments?sslmode=disable")
		intervalMS        = intFromEnv("INTERVAL_MS", 100)
		workers           = intFromEnv("WORKERS", 5)
		asset             = getEnv("ASSET", "USD/2")
		scheme            = getEnv("SCHEME", "test")
		typeStr           = getEnv("TYPE", "payin")
		encryptionKey     = getEnv("CONFIG_ENCRYPTION_KEY", "mysuperencryptionkey")
		encryptionOptions = "compress-algo=1, cipher-algo=aes256"
	)

	// Flags override env
	flag.StringVar(&dsn, "dsn", dsn, "Postgres DSN")
	flag.IntVar(&intervalMS, "interval-ms", intervalMS, "Delay between inserts per worker in ms")
	flag.IntVar(&workers, "workers", workers, "Number of concurrent workers")
	flag.StringVar(&asset, "asset", asset, "Asset string to set on payments (e.g. USD/2)")
	flag.StringVar(&scheme, "scheme", scheme, "Scheme string to set on payments (e.g. card)")
	flag.StringVar(&typeStr, "type", typeStr, "Payment type: payin|payout|transfer")
	flag.StringVar(&encryptionKey, "encryption-key", encryptionKey, "DB encryption key")

	// batching flags
	var batchSizeFlag = intFromEnv("BATCH_SIZE", 500)
	var flushMSFlag = intFromEnv("FLUSH_MS", 10000)
	flag.IntVar(&batchSizeFlag, "batch-size", batchSizeFlag, "Number of rows per INSERT (default 1)")
	flag.IntVar(&flushMSFlag, "flush-ms", flushMSFlag, "Max time to wait before flushing a partial batch in ms (default: interval-ms)")

	flag.Parse()

	pt, err := toPaymentType(typeStr)
	if err != nil {
		log.Fatalf("%v", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	connectorID, err := createConnector(db, ctx, encryptionKey, encryptionOptions)
	if err != nil {
		return
	}

	log.Printf("continuous-writer starting: workers=%d interval=%dms batch=%d flush=%dms connector=%s type=%s asset=%s scheme=%s", workers, intervalMS, batchSizeFlag, flushMSFlag, connectorID, pt, asset, scheme)
	interval := time.Duration(intervalMS) * time.Millisecond

	// Insert payments into public.payments table (schema-qualified)
	// We will build multi-row INSERT statements dynamically based on batch size.
	batchSize := batchSizeFlag
	flushMS := flushMSFlag // max time to wait before flushing a partial batch

	done := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer func() { done <- struct{}{} }()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(worker)))
			// buffers for batch
			buf := make([]paymentRow, 0, batchSize)
			flushInterval := time.Duration(flushMS) * time.Millisecond
			lastFlush := time.Now()
			for {
				// graceful shutdown check without waiting
				select {
				case <-ctx.Done():
					// flush remaining before exit
					if len(buf) > 0 {
						ctxIns, cancel := context.WithTimeout(context.Background(), 3*time.Second)
						err = flushBatch(ctxIns, db, buf, connectorID, pt, asset, scheme)
						if err != nil {
							log.Printf("unable to flush remaining items, err: %v", err)
						}
						cancel()
					}
					return
				default:
				}

				// generate one row per loop
				id := uuid.NewString()
				ref := fmt.Sprintf("loadgen-%d-%s", worker, id[:8])
				amount := r.Intn(10000) + 1 // 1..10000 minor units
				initAmount := amount
				buf = append(buf, paymentRow{
					id:         id,
					ref:        ref,
					initAmount: initAmount,
					amount:     amount,
				})

				shouldFlush := len(buf) >= batchSize || time.Since(lastFlush) >= flushInterval
				if shouldFlush {
					log.Printf("worker %d, bufSize bigger %t: flushTIme bigger %t", worker, len(buf) >= batchSize, time.Since(lastFlush) >= flushInterval)
					count := len(buf)
					ctxIns, cancel := context.WithTimeout(ctx, 5*time.Second)
					err := flushBatch(ctxIns, db, buf, connectorID, pt, asset, scheme)
					cancel()
					if err != nil {
						log.Printf("worker %d batch insert error (n=%d): %v", worker, count, err)
					} else {
						log.Printf("inserted batch w=%d n=%d last_id=%s", worker, count, buf[count-1].id)
					}
					buf = buf[:0]
					lastFlush = time.Now()
				}

				// If flush interval already elapsed, flush now without extra sleep to honor FLUSH_MS more tightly
				if len(buf) > 0 && time.Since(lastFlush) >= flushInterval {
					count := len(buf)
					ctxIns, cancel := context.WithTimeout(ctx, 5*time.Second)
					err := flushBatch(ctxIns, db, buf, connectorID, pt, asset, scheme)
					cancel()
					if err != nil {
						log.Printf("worker %d batch insert error (pre-sleep, n=%d): %v", worker, count, err)
					} else if worker == 0 {
						log.Printf("inserted batch (pre-sleep) n=%d last_id=%s", count, buf[count-1].id)
					}
					buf = buf[:0]
					lastFlush = time.Now()
				}

				// Sleep only after a flush to pace batches; otherwise keep generating.
				if shouldFlush {
					sleep := interval + time.Duration(r.Intn(50))*time.Millisecond
					timer := time.NewTimer(sleep)
					select {
					case <-ctx.Done():
						timer.Stop()
						// flush remaining before exit
						if len(buf) > 0 {
							ctxIns, cancel := context.WithTimeout(context.Background(), 3*time.Second)
							_ = flushBatch(ctxIns, db, buf, connectorID, pt, asset, scheme)
							cancel()
						}
						return
					case <-timer.C:
					}
				}
			}
		}(i)
	}

	// Wait for signal
	<-ctx.Done()
	log.Println("shutting down, waiting workers...")
	for i := 0; i < workers; i++ {
		<-done
	}
	log.Println("bye")
}

func createConnector(db *sql.DB, ctx context.Context, encryptionKey string, encryptionOptions string) (string, error) {
	// Optionally create/upsert connector entry
	defaultConnectorId := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "qonto",
	}
	connectorID := defaultConnectorId.String()
	upsert := `INSERT INTO "public"."connectors" (id, name, created_at, provider, reference)
		VALUES ($1, $2, now(), $3, $4)`
	_, err := db.ExecContext(ctx, upsert, connectorID, "test-connector"+connectorID, defaultConnectorId.Provider, defaultConnectorId.Reference)
	if err != nil {
		log.Printf("warning: failed to upsert connector '%s': %v", connectorID, err)
		return "", err
	} else {
		config := []byte(`{"ClientID": "asdf", "APIKey": "asdf", "Endpoint": "http://localhost:/"}`)
		update := `UPDATE public.connectors SET config=pgp_sym_encrypt($1::TEXT, $2, $3) WHERE id=$4` //, "{}", configEncryptionKey, encryptionOptions
		_, err = db.ExecContext(ctx, update, config, encryptionKey, encryptionOptions, connectorID)
		if err != nil {
			log.Printf("warning: failed to set config for connector '%s': %v", connectorID, err)
			return "", err
		} else {
			log.Printf("connector inserted: id=%s provider=%s reference=%s", connectorID, defaultConnectorId.Provider, defaultConnectorId.Reference)
		}
	}
	return connectorID, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func intFromEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func toPaymentType(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "payin":
		return "PAYIN", nil
	case "payout":
		return "PAYOUT", nil
	case "transfer":
		return "TRANSFER", nil
	default:
		return "", fmt.Errorf("unknown payment type: %s", s)
	}
}

func flushBatch(ctx context.Context, db *sql.DB, rows []paymentRow, connectorID string, pt string, asset, scheme string) error {
	if len(rows) == 0 {
		return nil
	}
	n := len(rows)
	placeholders := make([]string, 0, n)
	args := make([]any, 0, n*10)
	for j := 0; j < n; j++ {
		idx := j * 10
		placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,now(),$%d,$%d,$%d,$%d,$%d,$%d,$%d)", idx+1, idx+2, idx+3, idx+4, idx+5, idx+6, idx+7, idx+8, idx+9, idx+10))
		rw := rows[j]
		args = append(args, rw.id, connectorID, rw.ref, pt, rw.initAmount, rw.amount, asset, scheme, rw.source, rw.dest)
	}
	stmt := "INSERT INTO \"public\".\"payments\" (id, connector_id, reference, created_at, type, initial_amount, amount, asset, scheme, source_account_id, destination_account_id) VALUES " + strings.Join(placeholders, ",")
	_, err := db.ExecContext(ctx, stmt, args...)
	return err
}
