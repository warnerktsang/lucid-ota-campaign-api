package campaign

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/entity"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/log"
	"testing"
)

var errCRUD = errors.New("error crud")

const longName = "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"

func TestCreateCampaignRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		model     CreateCampaignRequest
		wantError bool
	}{
		{"success", CreateCampaignRequest{Name: "test", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"}, false},
		{"name required", CreateCampaignRequest{Name: "", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"}, true},
		{"name too long", CreateCampaignRequest{Name: longName, TargetVersion: "2.3.1", FleetID: "fleet-sf-01"}, true},
		{"target version required", CreateCampaignRequest{Name: "test", TargetVersion: "", FleetID: "fleet-sf-01"}, true},
		{"target version too long", CreateCampaignRequest{Name: "test", TargetVersion: "123456789012345678901234567890123", FleetID: "fleet-sf-01"}, true},
		{"fleet id required", CreateCampaignRequest{Name: "test", TargetVersion: "2.3.1", FleetID: ""}, true},
		{"fleet id too long", CreateCampaignRequest{Name: "test", TargetVersion: "2.3.1", FleetID: "1234567890123456789012345678901234567"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			assert.Equal(t, tt.wantError, err != nil)
		})
	}
}

func TestUpdateCampaignRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		model     UpdateCampaignRequest
		wantError bool
	}{
		{"success", UpdateCampaignRequest{Name: "test", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"}, false},
		{"name required", UpdateCampaignRequest{Name: "", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"}, true},
		{"name too long", UpdateCampaignRequest{Name: longName, TargetVersion: "2.3.1", FleetID: "fleet-sf-01"}, true},
		{"target version required", UpdateCampaignRequest{Name: "test", TargetVersion: "", FleetID: "fleet-sf-01"}, true},
		{"fleet id required", UpdateCampaignRequest{Name: "test", TargetVersion: "2.3.1", FleetID: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			assert.Equal(t, tt.wantError, err != nil)
		})
	}
}

func Test_service_CRUD(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService(&mockRepository{}, logger)

	ctx := context.Background()

	// initial count
	count, _ := s.Count(ctx)
	assert.Equal(t, 0, count)

	// successful creation
	campaign, err := s.Create(ctx, CreateCampaignRequest{Name: "test", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"})
	assert.Nil(t, err)
	assert.NotEmpty(t, campaign.ID)
	id := campaign.ID
	assert.Equal(t, "test", campaign.Name)
	assert.Equal(t, "2.3.1", campaign.TargetVersion)
	assert.Equal(t, "fleet-sf-01", campaign.FleetID)
	assert.NotEmpty(t, campaign.CreatedAt)
	assert.NotEmpty(t, campaign.UpdatedAt)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)

	// validation error in creation
	_, err = s.Create(ctx, CreateCampaignRequest{Name: "", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"})
	assert.NotNil(t, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)

	// unexpected error in creation
	_, err = s.Create(ctx, CreateCampaignRequest{Name: "error", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"})
	assert.Equal(t, errCRUD, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)

	_, _ = s.Create(ctx, CreateCampaignRequest{Name: "test2", TargetVersion: "2.3.1", FleetID: "fleet-sf-01"})

	// update
	campaign, err = s.Update(ctx, id, UpdateCampaignRequest{Name: "test updated", TargetVersion: "2.4.0", FleetID: "fleet-la-02"})
	assert.Nil(t, err)
	assert.Equal(t, "test updated", campaign.Name)
	assert.Equal(t, "2.4.0", campaign.TargetVersion)
	assert.Equal(t, "fleet-la-02", campaign.FleetID)
	_, err = s.Update(ctx, "none", UpdateCampaignRequest{Name: "test updated", TargetVersion: "2.4.0", FleetID: "fleet-la-02"})
	assert.NotNil(t, err)

	// validation error in update
	_, err = s.Update(ctx, id, UpdateCampaignRequest{Name: "", TargetVersion: "2.4.0", FleetID: "fleet-la-02"})
	assert.NotNil(t, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 2, count)

	// unexpected error in update
	_, err = s.Update(ctx, id, UpdateCampaignRequest{Name: "error", TargetVersion: "2.4.0", FleetID: "fleet-la-02"})
	assert.Equal(t, errCRUD, err)
	count, _ = s.Count(ctx)
	assert.Equal(t, 2, count)

	// get
	_, err = s.Get(ctx, "none")
	assert.NotNil(t, err)
	campaign, err = s.Get(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, "test updated", campaign.Name)
	assert.Equal(t, id, campaign.ID)

	// query
	campaigns, _ := s.Query(ctx, 0, 0)
	assert.Equal(t, 2, len(campaigns))

	// delete
	_, err = s.Delete(ctx, "none")
	assert.NotNil(t, err)
	campaign, err = s.Delete(ctx, id)
	assert.Nil(t, err)
	assert.Equal(t, id, campaign.ID)
	count, _ = s.Count(ctx)
	assert.Equal(t, 1, count)
}

type mockRepository struct {
	items []entity.Campaign
}

func (m mockRepository) Get(ctx context.Context, id string) (entity.Campaign, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return entity.Campaign{}, sql.ErrNoRows
}

func (m mockRepository) Count(ctx context.Context) (int, error) {
	return len(m.items), nil
}

func (m mockRepository) Query(ctx context.Context, offset, limit int) ([]entity.Campaign, error) {
	return m.items, nil
}

func (m *mockRepository) Create(ctx context.Context, campaign entity.Campaign) error {
	if campaign.Name == "error" {
		return errCRUD
	}
	m.items = append(m.items, campaign)
	return nil
}

func (m *mockRepository) Update(ctx context.Context, campaign entity.Campaign) error {
	if campaign.Name == "error" {
		return errCRUD
	}
	for i, item := range m.items {
		if item.ID == campaign.ID {
			m.items[i] = campaign
			break
		}
	}
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	for i, item := range m.items {
		if item.ID == id {
			m.items[i] = m.items[len(m.items)-1]
			m.items = m.items[:len(m.items)-1]
			break
		}
	}
	return nil
}
