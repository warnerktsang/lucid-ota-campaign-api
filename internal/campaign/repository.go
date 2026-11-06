package campaign

import (
	"context"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/entity"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/dbcontext"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/log"
)

// Repository encapsulates the logic to access campaigns from the data source.
type Repository interface {
	// Get returns the campaign with the specified campaign ID.
	Get(ctx context.Context, id string) (entity.Campaign, error)
	// Count returns the number of campaigns.
	Count(ctx context.Context) (int, error)
	// Query returns the list of campaigns with the given offset and limit.
	Query(ctx context.Context, offset, limit int) ([]entity.Campaign, error)
	// Create saves a new campaign in the storage.
	Create(ctx context.Context, campaign entity.Campaign) error
	// Update updates the campaign with given ID in the storage.
	Update(ctx context.Context, campaign entity.Campaign) error
	// Delete removes the campaign with given ID from the storage.
	Delete(ctx context.Context, id string) error
}

// repository persists campaigns in database
type repository struct {
	db     *dbcontext.DB
	logger log.Logger
}

// NewRepository creates a new campaign repository
func NewRepository(db *dbcontext.DB, logger log.Logger) Repository {
	return repository{db, logger}
}

// Get reads the campaign with the specified ID from the database.
func (r repository) Get(ctx context.Context, id string) (entity.Campaign, error) {
	var campaign entity.Campaign
	err := r.db.With(ctx).Select().Model(id, &campaign)
	return campaign, err
}

// Create saves a new campaign record in the database.
// It returns the ID of the newly inserted campaign record.
func (r repository) Create(ctx context.Context, campaign entity.Campaign) error {
	return r.db.With(ctx).Model(&campaign).Insert()
}

// Update saves the changes to a campaign in the database.
func (r repository) Update(ctx context.Context, campaign entity.Campaign) error {
	return r.db.With(ctx).Model(&campaign).Update()
}

// Delete deletes a campaign with the specified ID from the database.
func (r repository) Delete(ctx context.Context, id string) error {
	campaign, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	return r.db.With(ctx).Model(&campaign).Delete()
}

// Count returns the number of the campaign records in the database.
func (r repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.With(ctx).Select("COUNT(*)").From("campaign").Row(&count)
	return count, err
}

// Query retrieves the campaign records with the specified offset and limit from the database.
func (r repository) Query(ctx context.Context, offset, limit int) ([]entity.Campaign, error) {
	var campaigns []entity.Campaign
	err := r.db.With(ctx).
		Select().
		OrderBy("id").
		Offset(int64(offset)).
		Limit(int64(limit)).
		All(&campaigns)
	return campaigns, err
}
