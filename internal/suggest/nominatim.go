package suggest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"roadbook/internal/domain"
)

// Nominatim reverse-geocodes the destination against a Nominatim instance —
// by default the public openstreetmap.org one, whose usage policy allows
// light interactive use with an identifying User-Agent. One request per
// confirm click is well inside it. Opt-in via ROADBOOK_GEOCODER=nominatim;
// self-hosters can point BaseURL at their own instance.
type Nominatim struct {
	BaseURL string
	Client  *http.Client
}

func NewNominatim(baseURL string) *Nominatim {
	return &Nominatim{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// Suggest asks for the place at city zoom (10) and returns the most local
// named settlement. Failure to reach the geocoder is an error the caller
// reports; an address with no usable locality is an empty suggestion.
func (n *Nominatim) Suggest(ctx context.Context, dest domain.LatLng) (Suggestion, error) {
	u := fmt.Sprintf("%s/reverse?format=jsonv2&lat=%s&lon=%s&zoom=10&accept-language=en",
		n.BaseURL,
		url.QueryEscape(fmt.Sprintf("%.6f", dest.Lat)),
		url.QueryEscape(fmt.Sprintf("%.6f", dest.Lon)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Suggestion{}, err
	}
	// Nominatim's usage policy requires an identifying User-Agent.
	req.Header.Set("User-Agent", "roadbook/0.1 (self-hosted personal instance)")

	resp, err := n.Client.Do(req)
	if err != nil {
		return Suggestion{}, fmt.Errorf("nominatim: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Suggestion{}, fmt.Errorf("nominatim: HTTP %d", resp.StatusCode)
	}

	var body struct {
		Address map[string]string `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Suggestion{}, fmt.Errorf("nominatim: %w", err)
	}

	// Most-local first: the first present key names the settlement the
	// destination actually sits in; county and state catch rural areas.
	for _, k := range []string{"city", "town", "village", "municipality", "county", "state_district", "state"} {
		if v := body.Address[k]; v != "" {
			return Suggestion{Name: v, Source: "nominatim"}, nil
		}
	}
	return Suggestion{Source: "nominatim"}, nil
}
