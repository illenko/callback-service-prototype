package event

import (
	"context"
	"log"
	"log/slog"
	"testing"
	"time"

	"callback-service/internal/db"
	"callback-service/internal/event"
	"callback-service/internal/message"
	"callback-service/internal/payload"
	"callback-service/tests/testhelpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ProcessorTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	pool        *pgxpool.Pool
	repo        *db.CallbackRepository
	sut         *event.Processor
	ctx         context.Context
}

func (s *ProcessorTestSuite) SetupSuite() {
	time.Local = time.UTC

	s.ctx = context.Background()
	pgContainer, err := testhelpers.CreatePostgresContainer(s.ctx)
	if err != nil {
		log.Fatal(err)
	}
	s.pgContainer = pgContainer

	db.RunMigrations(pgContainer.ConnectionString, "../../../migrations")

	pool, err := db.GetPool(pgContainer.ConnectionString)
	if err != nil {
		log.Fatal(err)
	}

	s.pool = pool
	s.repo = db.NewCallbackRepository(pool)
	s.sut = event.NewProcessor(s.repo, slog.Default())
}

func (s *ProcessorTestSuite) TearDownSuite() {
	s.pool.Close()

	if err := s.pgContainer.Terminate(s.ctx); err != nil {
		log.Fatalf("error terminating postgres container: %s", err)
	}
}

func (s *ProcessorTestSuite) SetupTest() {
	_, err := s.pool.Exec(s.ctx, "DELETE FROM callback_message")
	if err != nil {
		log.Fatalf("error truncating callback_message table: %s", err)
	}
}

func (s *ProcessorTestSuite) TestProcess_Success() {
	t := s.T()

	event := message.PaymentEvent{
		ID: uuid.New(),
		Payload: payload.Payment{
			ID:          uuid.New(),
			Status:      "completed",
			CallbackUrl: "http://example.com/callback",
		},
	}

	err := s.sut.Process(s.ctx, event)
	assert.NoError(t, err)

	entity, err := s.repo.SelectByID(s.ctx, event.ID)
	assert.NoError(t, err)
	assert.NotNil(t, entity)
	assert.Equal(t, event.Payload.ID, entity.PaymentID)
	assert.Equal(t, event.Payload.CallbackUrl, entity.Url)
	assert.WithinDuration(t, time.Now(), *entity.ScheduledAt, time.Second)
}

func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}
