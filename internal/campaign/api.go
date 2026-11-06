package campaign

import (
	"github.com/go-ozzo/ozzo-routing/v2"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/errors"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/log"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/pagination"
	"net/http"
)

// RegisterHandlers sets up the routing of the HTTP handlers.
func RegisterHandlers(r *routing.RouteGroup, service Service, authHandler routing.Handler, logger log.Logger) {
	res := resource{service, logger}

	r.Get("/campaigns/<id>", res.get)
	r.Get("/campaigns", res.query)

	r.Use(authHandler)

	// the following endpoints require a valid JWT
	r.Post("/campaigns", res.create)
	r.Put("/campaigns/<id>", res.update)
	r.Delete("/campaigns/<id>", res.delete)
}

type resource struct {
	service Service
	logger  log.Logger
}

func (r resource) get(c *routing.Context) error {
	campaign, err := r.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	return c.Write(campaign)
}

func (r resource) query(c *routing.Context) error {
	ctx := c.Request.Context()
	count, err := r.service.Count(ctx)
	if err != nil {
		return err
	}
	pages := pagination.NewFromRequest(c.Request, count)
	campaigns, err := r.service.Query(ctx, pages.Offset(), pages.Limit())
	if err != nil {
		return err
	}
	pages.Items = campaigns
	return c.Write(pages)
}

func (r resource) create(c *routing.Context) error {
	var input CreateCampaignRequest
	if err := c.Read(&input); err != nil {
		r.logger.With(c.Request.Context()).Info(err)
		return errors.BadRequest("")
	}
	campaign, err := r.service.Create(c.Request.Context(), input)
	if err != nil {
		return err
	}

	return c.WriteWithStatus(campaign, http.StatusCreated)
}

func (r resource) update(c *routing.Context) error {
	var input UpdateCampaignRequest
	if err := c.Read(&input); err != nil {
		r.logger.With(c.Request.Context()).Info(err)
		return errors.BadRequest("")
	}

	campaign, err := r.service.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	return c.Write(campaign)
}

func (r resource) delete(c *routing.Context) error {
	campaign, err := r.service.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	return c.Write(campaign)
}
