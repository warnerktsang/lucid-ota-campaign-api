package campaign

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/entity"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/test"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/log"
	"testing"
	"time"
)

func TestRepository(t *testing.T) {
	logger, _ := log.NewForTest()
	db := test.DB(t)
	test.ResetTables(t, db, "campaign")
	repo := NewRepository(db, logger)

	ctx := context.Background()

	// initial count
	count, err := repo.Count(ctx)
	assert.Nil(t, err)

	// create
	err = repo.Create(ctx, entity.Campaign{
		ID:            "test1",
		Name:          "campaign1",
		TargetVersion: "2.3.1",
		FleetID:       "fleet-sf-01",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	assert.Nil(t, err)
	count2, _ := repo.Count(ctx)
	assert.Equal(t, 1, count2-count)

	// get
	campaign, err := repo.Get(ctx, "test1")
	assert.Nil(t, err)
	assert.Equal(t, "campaign1", campaign.Name)
	assert.Equal(t, "2.3.1", campaign.TargetVersion)
	assert.Equal(t, "fleet-sf-01", campaign.FleetID)
	_, err = repo.Get(ctx, "test0")
	assert.Equal(t, sql.ErrNoRows, err)

	// update
	err = repo.Update(ctx, entity.Campaign{
		ID:            "test1",
		Name:          "campaign1 updated",
		TargetVersion: "2.4.0",
		FleetID:       "fleet-sf-01",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	assert.Nil(t, err)
	campaign, _ = repo.Get(ctx, "test1")
	assert.Equal(t, "campaign1 updated", campaign.Name)
	assert.Equal(t, "2.4.0", campaign.TargetVersion)

	// query
	campaigns, err := repo.Query(ctx, 0, count2)
	assert.Nil(t, err)
	assert.Equal(t, count2, len(campaigns))

	// delete
	err = repo.Delete(ctx, "test1")
	assert.Nil(t, err)
	_, err = repo.Get(ctx, "test1")
	assert.Equal(t, sql.ErrNoRows, err)
	err = repo.Delete(ctx, "test1")
	assert.Equal(t, sql.ErrNoRows, err)
}
