package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/db/migrations"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/internal/api"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/internal/events/publishers"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/internal/repository/postgres"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/oapi"
)

// recordingPublisher captures published events for assertions. Used by every
// HTTP integration test in this package — mirrors the convention used in
// athlete-svc's testcontainers fixture: tests don't mock; they capture.
type recordingPublisher struct {
	events []publishers.CampEvent
}

func (r *recordingPublisher) Publish(_ context.Context, ev publishers.CampEvent) error {
	r.events = append(r.events, ev)
	return nil
}

func newTestServer(t *testing.T) (http.Handler, *recordingPublisher) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	ctx := t.Context()
	pgC, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("api_svc"),
		tcpostgres.WithUsername("emc"),
		tcpostgres.WithPassword("emc"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(pgC) })

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Run migrations against the fresh container.
	goose.SetBaseFS(migrations.Files)
	require.NoError(t, goose.SetDialect("postgres"))
	db, err := goose.OpenDBWithDriver("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, goose.UpContext(ctx, db, "."))
	require.NoError(t, db.Close())

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	pub := &recordingPublisher{}
	srv := api.NewServer(postgres.New(pool), pub)

	r := gin.New()
	oapi.RegisterHandlers(r, oapi.NewStrictHandler(srv, nil))
	return r, pub
}

func TestCreateAndGetCamp_Roundtrip(t *testing.T) {
	r, pub := newTestServer(t)

	body := []byte(`{
		"name": "Summer Football Camp",
		"sport": "football",
		"location": "Bradenton, FL",
		"capacity": 60,
		"start_date": "2026-06-15",
		"end_date":   "2026-06-22"
	}`)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/camps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "POST /camps body: %s", rec.Body.String())

	var created oapi.Camp
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.Equal(t, "Summer Football Camp", created.Name)
	require.Equal(t, "football", created.Sport)
	require.Equal(t, 60, created.Capacity)

	// Verify exactly one event was published with matching payload.
	require.Len(t, pub.events, 1)
	require.Equal(t, created.Id.String(), pub.events[0].ID)
	require.WithinDuration(t, time.Now(), pub.events[0].OccurredAt, 5*time.Second)

	// Round-trip through GET /camps/{id}.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/camps/%s", created.Id.String()), nil)
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "GET body: %s", rec.Body.String())

	var fetched oapi.Camp
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fetched))
	require.Equal(t, created.Id, fetched.Id)
	require.Equal(t, created.Name, fetched.Name)
}

func TestGetCamp_NotFound(t *testing.T) {
	r, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/camps/00000000-0000-0000-0000-000000000000", nil)
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
