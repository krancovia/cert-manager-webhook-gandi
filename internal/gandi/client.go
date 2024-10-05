package gandi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const ttl = 300 // The minimum TTL allowed by Gandi

type resourceRecordSet struct {
	Type   string   `json:"rrset_type"`
	TTL    int      `json:"rrset_ttl"`
	Name   string   `json:"rrset_name"`
	Values []string `json:"rrset_values"`
}

type client struct {
	baseURL     string // Overridable for testing purposes
	accessToken string
	client      *http.Client
}

func newClient(accessToken string) *client {
	return &client{
		baseURL:     "https://dns.api.gandi.net/api/v5",
		accessToken: accessToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *client) getTxtRecordValues(domain string, name string) ([]string, error) {
	// GET <API BASE URL>/domains/<DOMAIN>/records/<NAME>/TXT
	req, err := http.NewRequest(http.MethodGet, c.txtRecordURL(domain, name), nil)
	if err != nil {
		return nil, fmt.Errorf("error building LiveDNS API request: %w", err)
	}
	status, body, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error executing Live DNS API request: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status == http.StatusOK {
		rrs := &resourceRecordSet{}
		if err := json.Unmarshal(body, rrs); err != nil {
			return nil, fmt.Errorf("error unmarshaling resource record set from JSON: %w", err)
		}
		for i := range rrs.Values {
			rrs.Values[i] = strings.Trim(rrs.Values[i], `"`)
		}
		return rrs.Values, nil
	}
	return nil, fmt.Errorf(
		"unexpected HTTP status in response to Live DNS API request: %d",
		status,
	)
}

func (c *client) createTxtRecord(domain, name string, values []string) error {
	// POST <API BASE URL>/domains/<DOMAIN>/records
	body, err := json.Marshal(
		resourceRecordSet{
			Type:   "TXT",
			TTL:    ttl,
			Name:   name,
			Values: values,
		},
	)
	if err != nil {
		return fmt.Errorf("error marshaling resource record set to JSON: %w", err)
	}
	req, err := http.NewRequest(
		http.MethodPost,
		c.recordsURL(domain),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("error building LiveDNS API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	status, _, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing LiveDNS API request: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf(
			"unexpected HTTP status in response to Live DNS API request: %d",
			status,
		)
	}
	return nil
}

func (c *client) updateTxtRecord(domain, name string, values []string) error {
	// PUT <API BASE URL>/domains/<DOMAIN>/records/<NAME>/TXT
	body, err := json.Marshal(
		struct {
			TTL    int      `json:"rrset_ttl"`
			Values []string `json:"rrset_values"`
		}{
			TTL:    ttl,
			Values: values,
		},
	)
	if err != nil {
		return fmt.Errorf("error marshaling resource record set to JSON: %w", err)
	}
	req, err := http.NewRequest(
		http.MethodPut,
		c.txtRecordURL(domain, name),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("error building LiveDNS API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	status, _, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing LiveDNS API request: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf(
			"unexpected HTTP status in response to Live DNS API request: %d",
			status,
		)
	}
	return nil
}

func (c *client) deleteTxtRecord(domain, name string) error {
	// DELETE <API BASE URL>/domains/<DOMAIN>/records/<NAME>/TXT
	req, err := http.NewRequest(
		http.MethodDelete,
		c.txtRecordURL(domain, name),
		nil,
	)
	if err != nil {
		return fmt.Errorf("error building LiveDNS API request: %w", err)
	}
	status, _, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("error executing LiveDNS API request: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf(
			"unexpected HTTP status in response to Live DNS API request: %d",
			status,
		)
	}
	return nil
}

func (c *client) txtRecordURL(domain, name string) string {
	return fmt.Sprintf("%s/%s/TXT", c.recordsURL(domain), name)
}

func (c *client) recordsURL(domain string) string {
	return fmt.Sprintf("%s/domains/%s/records", c.baseURL, domain)
}

func (c *client) doRequest(req *http.Request) (int, []byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	res, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("error reading response body: %w", err)
	}
	return res.StatusCode, bodyBytes, nil
}
