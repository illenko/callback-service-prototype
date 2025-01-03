package customer

import (
	"context"
	"log"
	"testing"
	"time"

	"callback-service/internal/db"
	"callback-service/tests/testhelpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CallbackRepositoryTestSuite struct {
	suite.Suite
	pgContainer *testhelpers.PostgresContainer
	pool        *pgxpool.Pool
	sut         *db.CallbackRepository
	ctx         context.Context
}

func (s *CallbackRepositoryTestSuite) SetupSuite() {
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
	s.sut = db.NewCallbackRepository(pool)
}

func (s *CallbackRepositoryTestSuite) TearDownSuite() {
	s.pool.Close()

	if err := s.pgContainer.Terminate(s.ctx); err != nil {
		log.Fatalf("error terminating postgres container: %s", err)
	}
}

func (s *CallbackRepositoryTestSuite) SetupTest() {
	_, err := s.pool.Exec(s.ctx, "DELETE FROM callback_message")
	if err != nil {
		log.Fatalf("error truncating callback_message table: %s", err)
	}
}

func (s *CallbackRepositoryTestSuite) TestBeginTx() {
	t := s.T()

	tx, err := s.sut.BeginTx(s.ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	err = tx.Rollback(s.ctx)
	assert.NoError(t, err)
}

func (s *CallbackRepositoryTestSuite) TestCreate() {
	t := s.T()

	now := time.Now()
	payload := `{"key": "value"}`

	entity := &db.CallbackMessageEntity{
		ID:               uuid.New(),
		PaymentID:        uuid.New(),
		Url:              "http://example.com",
		Payload:          payload,
		ScheduledAt:      &now,
		DeliveryAttempts: 0,
	}

	createdEntity, err := s.sut.Create(s.ctx, entity)
	assert.NoError(t, err)
	assert.NotNil(t, createdEntity)
	assert.Equal(t, entity.ID, createdEntity.ID)
}

func (s *CallbackRepositoryTestSuite) TestGetUnprocessedCallbacks() {
	t := s.T()

	past := time.Now().Add(-time.Hour)
	payload := `{"key": "value"}`

	entity := &db.CallbackMessageEntity{
		ID:               uuid.New(),
		PaymentID:        uuid.New(),
		Url:              "http://example.com",
		Payload:          payload,
		ScheduledAt:      &past,
		DeliveryAttempts: 0,
	}
	_, err := s.sut.Create(s.ctx, entity)
	assert.NoError(t, err)

	tx, err := s.sut.BeginTx(s.ctx)
	assert.NoError(t, err)
	defer tx.Rollback(s.ctx)

	callbacks, err := s.sut.GetUnprocessedCallbacks(s.ctx, tx, 10)
	assert.NoError(t, err)
	assert.NotEmpty(t, callbacks)
	assert.Equal(t, entity.ID, callbacks[0].ID)
}

func (s *CallbackRepositoryTestSuite) TestUpdate() {
	t := s.T()

	tx, err := s.sut.BeginTx(s.ctx)
	assert.NoError(t, err)
	defer tx.Rollback(s.ctx)

	now := time.Now()
	payload := `{"key": "value"}`

	entity := &db.CallbackMessageEntity{
		ID:               uuid.New(),
		PaymentID:        uuid.New(),
		Url:              "http://example.com",
		Payload:          payload,
		ScheduledAt:      &now,
		DeliveryAttempts: 0,
	}

	_, err = s.sut.Create(s.ctx, entity)
	assert.NoError(t, err)

	entity.DeliveryAttempts = 1
	err = s.sut.Update(s.ctx, tx, entity)
	assert.NoError(t, err)

	updatedEntity, err := s.sut.SelectForUpdateByID(s.ctx, tx, entity.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedEntity.DeliveryAttempts)
}

func (s *CallbackRepositoryTestSuite) TestSelectForUpdateByID() {
	t := s.T()

	tx, err := s.sut.BeginTx(s.ctx)
	assert.NoError(t, err)
	defer tx.Rollback(s.ctx)

	now := time.Now()
	payload := `{"key": "value"}`

	entity := &db.CallbackMessageEntity{
		ID:               uuid.New(),
		PaymentID:        uuid.New(),
		Url:              "http://example.com",
		Payload:          payload,
		ScheduledAt:      &now,
		DeliveryAttempts: 0,
	}

	_, err = s.sut.Create(s.ctx, entity)
	assert.NoError(t, err)

	selectedEntity, err := s.sut.SelectForUpdateByID(s.ctx, tx, entity.ID)
	assert.NoError(t, err)
	assert.Equal(t, entity.ID, selectedEntity.ID)
}

func TestCallbackRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(CallbackRepositoryTestSuite))
}
