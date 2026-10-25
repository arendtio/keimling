package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func isOpenRouter(base string) bool {
	u, err := url.Parse(base)
	return err == nil && strings.EqualFold(u.Hostname(), "openrouter.ai")
}

func endpointDisplay(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<invalid URL>"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// CheckModel validates an identifier against the provider's model catalog.
// A missing model is an error; a catalog that cannot be read is a warning at startup.
func (c *Client) CheckModel(ctx context.Context, model string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("models"), nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("model catalog unavailable: %w", err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 12<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("model catalog HTTP %d at %s", res.StatusCode, endpointDisplay(req.URL.String()))
	}
	var doc struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err = json.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("invalid model catalog: %w", err)
	}
	if doc.Data == nil {
		return fmt.Errorf("model catalog missing data array")
	}
	ids := make([]string, 0, len(doc.Data))
	for _, item := range doc.Data {
		ids = append(ids, item.ID)
	}
	return validateModelInCatalog(ids, model, isOpenRouter(c.BaseURL))
}

// validateModelInCatalog reports whether model is usable. OpenRouter routing
// suffixes such as ":floor", ":nitro" or ":exacto" are not catalog IDs, so for
// OpenRouter the base model is validated while the suffix is kept in requests.
func validateModelInCatalog(ids []string, model string, openRouter bool) error {
	for _, id := range ids {
		if id == model {
			return nil
		}
	}
	if openRouter {
		if base, ok := openRouterBaseModel(model); ok {
			for _, id := range ids {
				if id == base {
					return nil
				}
			}
			return fmt.Errorf("model %q (base of %q) not found in provider catalog", base, model)
		}
	}
	return fmt.Errorf("model %q not found in provider catalog", model)
}

// openRouterBaseModel strips a routing suffix (":floor", ":nitro", ":exacto",
// ":free", ...) from an OpenRouter model identifier.
func openRouterBaseModel(model string) (string, bool) {
	i := strings.LastIndex(model, ":")
	if i <= 0 || i == len(model)-1 {
		return "", false
	}
	return model[:i], true
}

func performStartupCheck(c *Client, model string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	return c.CheckModel(ctx, model)
}
