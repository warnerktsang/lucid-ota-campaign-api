package campaign

import (
	"context"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/entity"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/log"
	"time"
)

// Service encapsulates usecase logic for campaigns.
type Service interface {
	Get(ctx context.Context, id string) (Campaign, error)
	Query(ctx context.Context, offset, limit int) ([]Campaign, error)
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, input CreateCampaignRequest) (Campaign, error)
	Update(ctx context.Context, id string, input UpdateCampaignRequest) (Campaign, error)
	Delete(ctx context.Context, id string) (Campaign, error)
}

// Campaign represents the data about a campaign.
type Campaign struct {
	entity.Campaign
}

// CreateCampaignRequest represents a campaign creation request.
type CreateCampaignRequest struct {
	Name          string `json:"name"`
	TargetVersion string `json:"target_version"`
	FleetID       string `json:"fleet_id"`
}

// Validate validates the CreateCampaignRequest fields.
func (m CreateCampaignRequest) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.Name, validation.Required, validation.Length(0, 128)),
		validation.Field(&m.TargetVersion, validation.Required, validation.Length(0, 32)),
		validation.Field(&m.FleetID, validation.Required, validation.Length(0, 36)),
	)
}

// UpdateCampaignRequest represents a campaign update request.
type UpdateCampaignRequest struct {
	Name          string `json:"name"`
	TargetVersion string `json:"target_version"`
	FleetID       string `json:"fleet_id"`
}

// Validate validates the UpdateCampaignRequest fields.
func (m UpdateCampaignRequest) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.Name, validation.Required, validation.Length(0, 128)),
		validation.Field(&m.TargetVersion, validation.Required, validation.Length(0, 32)),
		validation.Field(&m.FleetID, validation.Required, validation.Length(0, 36)),
	)
}

type service struct {
	repo   Repository
	logger log.Logger
}

// NewService creates a new campaign service.
func NewService(repo Repository, logger log.Logger) Service {
	return service{repo, logger}
}

// Get returns the campaign with the specified the campaign ID.
func (s service) Get(ctx context.Context, id string) (Campaign, error) {
	campaign, err := s.repo.Get(ctx, id)
	if err != nil {
		return Campaign{}, err
	}
	return Campaign{campaign}, nil
}

// Create creates a new campaign.
func (s service) Create(ctx context.Context, req CreateCampaignRequest) (Campaign, error) {
	if err := req.Validate(); err != nil {
		return Campaign{}, err
	}
	id := entity.GenerateID()
	now := time.Now()
	err := s.repo.Create(ctx, entity.Campaign{
		ID:            id,
		Name:          req.Name,
		TargetVersion: req.TargetVersion,
		FleetID:       req.FleetID,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return Campaign{}, err
	}
	return s.Get(ctx, id)
}

// Update updates the campaign with the specified ID.
func (s service) Update(ctx context.Context, id string, req UpdateCampaignRequest) (Campaign, error) {
	if err := req.Validate(); err != nil {
		return Campaign{}, err
	}

	campaign, err := s.Get(ctx, id)
	if err != nil {
		return campaign, err
	}
	campaign.Name = req.Name
	campaign.TargetVersion = req.TargetVersion
	campaign.FleetID = req.FleetID
	campaign.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, campaign.Campaign); err != nil {
		return campaign, err
	}
	return campaign, nil
}

// Delete deletes the campaign with the specified ID.
func (s service) Delete(ctx context.Context, id string) (Campaign, error) {
	campaign, err := s.Get(ctx, id)
	if err != nil {
		return Campaign{}, err
	}
	if err = s.repo.Delete(ctx, id); err != nil {
		return Campaign{}, err
	}
	return campaign, nil
}

// Count returns the number of campaigns.
func (s service) Count(ctx context.Context) (int, error) {
	return s.repo.Count(ctx)
}

// Query returns the campaigns with the specified offset and limit.
func (s service) Query(ctx context.Context, offset, limit int) ([]Campaign, error) {
	items, err := s.repo.Query(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	result := []Campaign{}
	for _, item := range items {
		result = append(result, Campaign{item})
	}
	return result, nil
}
