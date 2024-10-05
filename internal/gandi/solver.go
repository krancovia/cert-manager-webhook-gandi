package gandi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// solver is an implementation of the webhook.Solver interface that solves ACME
// DNS-01 challenges using the Gandi LiveDNS API.
type solver struct {
	client    kubernetes.Interface
	zoneMusMu sync.Mutex
	zoneMus   map[string]*sync.Mutex
}

// NewSolver returns an implementation of the webhook.Solver interface that
// solves ACME DNS-01 challenges using the Gandi LiveDNS API.
func NewSolver() webhook.Solver {
	return &solver{
		zoneMus: map[string]*sync.Mutex{},
	}
}

// Name implements the webhook.Solver interface.
func (s *solver) Name() string {
	return "gandi"
}

// Initialize implements the webhook.Solver interface.
func (s *solver) Initialize(restCfg *rest.Config, _ <-chan struct{}) error {
	// By not setting this here if it's already been set, we allow for the
	// possibility of injecting a fake clientset for testing purposes while still
	// allowing a client to be constructed from the provided rest.Config
	// otherwise.
	if s.client != nil {
		return nil
	}
	cl, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("unable to get k8s client: %v", err)
	}
	s.client = cl
	return nil
}

// Present implements the webhook.Solver interface.
func (s *solver) Present(cr *v1alpha1.ChallengeRequest) error {
	cl, err := s.getClient(*cr)
	if err != nil {
		err = fmt.Errorf("error getting Gandi LiveDNS API client: %w", err)
		log.Println(err.Error())
		return err
	}
	zone, entry := s.getZoneAndEntry(*cr)
	s.getZoneLock(zone)
	defer s.releaseZoneLock(zone)
	values, err := cl.getTxtRecordValues(zone, entry)
	if err != nil {
		err = fmt.Errorf("error checking for existence of TXT record: %w", err)
		log.Println(err.Error())
		return err
	}
	if len(values) == 0 {
		if err = cl.createTxtRecord(zone, entry, []string{cr.Key}); err != nil {
			err = fmt.Errorf("error creating TXT record: %w", err)
			log.Println(err.Error())
			return err
		}
		return nil
	}
	values = append(values, cr.Key)
	if err = cl.updateTxtRecord(zone, entry, values); err != nil {
		err = fmt.Errorf("error updating TXT record: %w", err)
		log.Println(err.Error())
		return err
	}
	return nil
}

// CleanUp implements the webhook.Solver interface.
func (s *solver) CleanUp(cr *v1alpha1.ChallengeRequest) error {
	cl, err := s.getClient(*cr)
	if err != nil {
		err = fmt.Errorf("error getting Gandi LiveDNS API client: %w", err)
		log.Println(err.Error())
		return err
	}
	zone, entry := s.getZoneAndEntry(*cr)
	s.getZoneLock(zone)
	defer s.releaseZoneLock(zone)
	values, err := cl.getTxtRecordValues(zone, entry)
	if err != nil {
		err = fmt.Errorf("error checking for existence of TXT record: %w", err)
		log.Println(err.Error())
		return err
	}
	if len(values) == 0 {
		return nil
	}
	if len(values) == 1 {
		if err = cl.deleteTxtRecord(zone, entry); err != nil {
			err = fmt.Errorf("error deleting TXT record: %w", err)
			log.Println(err.Error())
			return err
		}
	}
	values = slices.DeleteFunc(values, func(val string) bool {
		return val == cr.Key
	})
	if err = cl.updateTxtRecord(zone, entry, values); err != nil {
		err = fmt.Errorf("error updating TXT record: %w", err)
		log.Println(err.Error())
		return err
	}
	return nil
}

// TODO: Add tests
func (s *solver) getZoneAndEntry(cr v1alpha1.ChallengeRequest) (string, string) {
	// Trim the zone off the end of the FQDN to get the entry
	entry := strings.TrimSuffix(cr.ResolvedFQDN, cr.ResolvedZone)
	// Both cr.ResolvedZone and entry will now with a '.'
	return strings.TrimSuffix(cr.ResolvedZone, "."), strings.TrimSuffix(entry, ".")
}

// getClient returns a new Gandi LiveDNS API client.
func (s *solver) getClient(cr v1alpha1.ChallengeRequest) (*client, error) {
	accessToken, err := s.getAccessToken(cr)
	if err != nil {
		return nil, err
	}
	return newClient(accessToken), nil
}

// getAccessToken gets a PAT for the Gandi LiveDNS from a Kubernetes Secret.
//
// TODO: Add tests
func (s *solver) getAccessToken(cr v1alpha1.ChallengeRequest) (string, error) {
	cfg := struct {
		APIKeySecretRef cmmeta.SecretKeySelector `json:"apiKeySecretRef"`
	}{}
	if err := json.Unmarshal(cr.Config.Raw, &cfg); err != nil {
		return "", fmt.Errorf("error decoding solver config: %w", err)
	}
	secretName := cfg.APIKeySecretRef.LocalObjectReference.Name
	secret, err := s.client.CoreV1().Secrets(cr.ResourceNamespace).Get(
		context.Background(),
		secretName,
		metav1.GetOptions{},
	)
	if err != nil {
		return "", fmt.Errorf(
			"error getting Secret %q in namespace %q: %w",
			secretName, cr.ResourceNamespace, err,
		)
	}
	apiKey := string(secret.Data[cfg.APIKeySecretRef.Key])
	if apiKey == "" {
		return "", fmt.Errorf(
			"key %q not found in secret \"%s/%s\"",
			cfg.APIKeySecretRef.Key, secretName, cr.ResourceNamespace)
	}
	return apiKey, nil
}

func (s *solver) getZoneLock(zone string) {
	// Look for a zone-specific mutex
	if zoneMu, exists := s.zoneMus[zone]; exists {
		// It exists, so lock it and return
		zoneMu.Lock()
		return
	}
	// The zone-specific mutex doesn't exist, so create it. This requires first
	// locking the the master mutex to ensure that only one goroutine is creating
	// the zone-specific mutex.
	s.zoneMusMu.Lock()
	defer s.zoneMusMu.Unlock()
	// Double-check that the zone-specific mutex doesn't exist in case another
	// goroutine created it while we were waiting for a lock on the master mutex.
	if zoneLock, exists := s.zoneMus[zone]; exists {
		zoneLock.Lock()
		return
	}
	// The zone-specific mutex doesn't exist, so create it and lock it.
	s.zoneMus[zone] = &sync.Mutex{}
	s.zoneMus[zone].Lock()
}

func (s *solver) releaseZoneLock(zone string) {
	if zoneMu, exists := s.zoneMus[zone]; exists {
		zoneMu.Unlock()
	}
}
