// Package api implements the HTTP server for api-svc.
package api

import (
	"context"
	"errors"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imgacademy/em-challenge/apps/api-svc/internal/events/publishers"
	"github.com/imgacademy/em-challenge/apps/api-svc/internal/repository/postgres"
	"github.com/imgacademy/em-challenge/apps/api-svc/oapi"
)

// EventPublisher is the subset of publishers.CampPublisher that the API layer
// uses. Defining it here lets tests inject a no-op publisher.
type EventPublisher interface {
	Publish(ctx context.Context, ev publishers.CampEvent) error
}

// Server implements oapi.StrictServerInterface.
type Server struct {
	q   *postgres.Queries
	pub EventPublisher
	now func() time.Time
}

// NewServer returns a Server. Pass a clock function to make tests deterministic.
func NewServer(q *postgres.Queries, pub EventPublisher) *Server {
	return &Server{q: q, pub: pub, now: time.Now}
}

func (s *Server) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// --- conversions between pgtype, oapi, and avro shapes ----------------------

func toAPICamp(c postgres.Camp) oapi.Camp {
	return oapi.Camp{
		Id:        openapi_types.UUID(c.ID.Bytes),
		Name:      c.Name,
		Sport:     c.Sport,
		Location:  c.Location,
		Capacity:  int(c.Capacity),
		StartDate: openapi_types.Date{Time: c.StartDate.Time},
		EndDate:   openapi_types.Date{Time: c.EndDate.Time},
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}

func toCampEvent(c postgres.Camp, occurredAt time.Time) publishers.CampEvent {
	return publishers.CampEvent{
		ID:         uuid.UUID(c.ID.Bytes).String(),
		Name:       c.Name,
		Sport:      c.Sport,
		Location:   c.Location,
		Capacity:   c.Capacity,
		StartDate:  c.StartDate.Time,
		EndDate:    c.EndDate.Time,
		OccurredAt: occurredAt,
	}
}

func pgDate(d openapi_types.Date) pgtype.Date {
	return pgtype.Date{Time: d.Time, Valid: true}
}

// --- handlers ----------------------------------------------------------------

// CreateCamp inserts a camp and publishes Camp.V1.
func (s *Server) CreateCamp(ctx context.Context, req oapi.CreateCampRequestObject) (oapi.CreateCampResponseObject, error) {
	body := req.Body
	camp, err := s.q.CreateCamp(ctx, postgres.CreateCampParams{
		Name:      body.Name,
		Sport:     body.Sport,
		Location:  body.Location,
		Capacity:  int32(body.Capacity), //nolint:gosec
		StartDate: pgDate(body.StartDate),
		EndDate:   pgDate(body.EndDate),
	})
	if err != nil {
		return nil, err
	}
	if err := s.pub.Publish(ctx, toCampEvent(camp, s.clock())); err != nil {
		return nil, err
	}
	resp := oapi.CreateCamp201JSONResponse(toAPICamp(camp))
	return resp, nil
}

// GetCamp returns a single camp or 404.
func (s *Server) GetCamp(ctx context.Context, req oapi.GetCampRequestObject) (oapi.GetCampResponseObject, error) {
	camp, err := s.q.GetCamp(ctx, pgtype.UUID{Bytes: req.Id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return oapi.GetCamp404JSONResponse{Code: "not_found", Message: "camp not found"}, nil
		}
		return nil, err
	}
	resp := oapi.GetCamp200JSONResponse(toAPICamp(camp))
	return resp, nil
}

// ListCamps returns paginated camps, optionally filtered by sport.
func (s *Server) ListCamps(ctx context.Context, req oapi.ListCampsRequestObject) (oapi.ListCampsResponseObject, error) {
	limit := int32(50)
	if req.Params.Limit != nil {
		limit = int32(*req.Params.Limit) //nolint:gosec
	}
	offset := int32(0)
	if req.Params.Offset != nil {
		offset = int32(*req.Params.Offset) //nolint:gosec
	}
	rows, err := s.q.ListCamps(ctx, postgres.ListCampsParams{
		Sport:  req.Params.Sport,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	items := make([]oapi.Camp, 0, len(rows))
	for _, r := range rows {
		items = append(items, toAPICamp(r))
	}
	return oapi.ListCamps200JSONResponse{
		Items:  items,
		Limit:  int(limit),
		Offset: int(offset),
	}, nil
}
