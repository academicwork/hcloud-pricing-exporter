package fetcher

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

var _ Fetcher = &server{}

// NewServer creates a new fetcher that will collect pricing information on servers.
func NewServer(pricing *PriceProvider, additionalLabels ...string) Fetcher {
	return &server{newBase(pricing, "server", []string{"location", "type"}, additionalLabels...)}
}

type server struct {
	*baseFetcher
}

func getServer(client *hcloud.Client) ([]*hcloud.Server, error) {
	page := 0
	var result []*hcloud.Server
	for {

		servers, response, err := client.Server.List(ctx, hcloud.ServerListOpts{
			ListOpts: hcloud.ListOpts{
				Page:    page,
				PerPage: 50,
			},
		})
		if err != nil {
			return nil, err
		}
		result = append(result, servers...)
		if page == response.Meta.Pagination.LastPage {
			break
		}
		page++
	}
	return result, nil
}

func (server server) Run(client *hcloud.Client) error {
	servers, err := getServer(client)
	if err != nil {
		return err
	}

	for _, s := range servers {
		location := s.Datacenter.Location

		labels := append([]string{
			s.Name,
			location.Name,
			s.ServerType.Name,
		},
			parseAdditionalLabels(server.additionalLabels, s.Labels)...,
		)
		pricing, err := findServerPricing(location, s.ServerType.Pricings)
		if err != nil {
			return err
		}

		parseToGauge(server.hourly.WithLabelValues(labels...), pricing.Hourly.Gross)
		parseToGauge(server.monthly.WithLabelValues(labels...), pricing.Monthly.Gross)
	}

	return nil
}

func findServerPricing(location *hcloud.Location, pricings []hcloud.ServerTypeLocationPricing) (*hcloud.ServerTypeLocationPricing, error) {
	for _, pricing := range pricings {
		if pricing.Location.Name == location.Name {
			return &pricing, nil
		}
	}

	return nil, fmt.Errorf("no server pricing found for location %s", location.Name)
}

