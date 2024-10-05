//go:build integration
// +build integration

package gandi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	whapi "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/issuer/acme/dns/util"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSolver(t *testing.T) {
	testNameServer := os.Getenv("TEST_NAME_SERVER")
	if testNameServer == "" {
		// Using one of Gandi's own DNS servers seems like a sensible default for
		// minimizing the time spent waiting for records and record deletions to
		// propagate.
		//
		// As of 2024-10-12, any of the following should work:
		//
		// TODO: Do we really need to include port numbers?
		testNameServer = "ns-22-a.gandi.net:53"
		// testNameServer = "ns-138-b.gandi.net:53"
		// testNameServer = "ns-146-c.gandi.net:53"
	}

	// Must be a zone that is managed by Gandi DNS
	testZone := os.Getenv("TEST_ZONE")
	require.NotEmpty(t, testZone)
	require.True(t, strings.HasSuffix(testZone, "."))

	testFQDN := fmt.Sprintf("cert-manager-dns01-tests.%s", testZone)

	// The DNS name for which we are simulating a challenge. MUST be within the
	// zone specified by testZone.
	testDNSName := os.Getenv("TEST_DNS_NAME")
	require.NotEmpty(t, testDNSName)
	require.True(
		t,
		strings.HasSuffix(
			testDNSName,
			strings.TrimSuffix(testZone, "."),
		),
	)

	testGandiPAT := os.Getenv("GANDI_PAT")
	require.NotEmpty(t, testGandiPAT)

	const testAccessTokenSecretNamespace = "cert-manager"
	const testAccessTokenSecretName = "gandi-access-token"
	const testAccessTokenSecretKey = "token"

	testJSONConfig := &apiextensionsv1.JSON{}
	var err error
	testJSONConfig.Raw, err = json.Marshal(map[string]any{
		"apiKeySecretRef": map[string]any{
			"name": testAccessTokenSecretName,
			"key":  testAccessTokenSecretKey,
		},
	})
	require.NoError(t, err)

	s := NewSolver()
	solver, ok := s.(*solver)
	require.True(t, ok)
	solver.client = fake.NewClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testAccessTokenSecretNamespace,
				Name:      testAccessTokenSecretName,
			},
			Data: map[string][]byte{
				testAccessTokenSecretKey: []byte(testGandiPAT),
			},
		},
	)

	ch1 := &whapi.ChallengeRequest{
		ResourceNamespace: testAccessTokenSecretNamespace,
		ResolvedFQDN:      testFQDN,
		ResolvedZone:      testZone,
		Config:            testJSONConfig,
		DNSName:           testDNSName,
		Key:               randomString(),
	}
	t.Logf("Presenting first ChallengeRequest: %#v", ch1)
	if err := solver.Present(ch1); err != nil {
		t.Errorf("expected Present to not error, but got: %v", err)
		return
	}
	defer func() {
		if err := solver.CleanUp(ch1); err != nil {
			t.Errorf("expected CleanUp to not error, but got: %v", err)
		}
	}()

	ch2 := &whapi.ChallengeRequest{
		ResourceNamespace: testAccessTokenSecretNamespace,
		ResolvedFQDN:      testFQDN,
		ResolvedZone:      testZone,
		Config:            testJSONConfig,
		DNSName:           testDNSName,
		Key:               randomString(),
	}
	t.Logf("Presenting second ChallengeRequest: %#v", ch2)
	if err := solver.Present(ch2); err != nil {
		t.Errorf("expected Present to not error, but got: %v", err)
		return
	}
	defer func() {
		if err := solver.CleanUp(ch2); err != nil {
			t.Errorf("expected CleanUp to not error, but got: %v", err)
		}
	}()

	pollInterval := time.Second * 3
	propagationLimit := time.Minute * 5

	t.Log("Waiting for both records to propagate...")
	if err := wait.PollUntilContextTimeout(
		context.Background(),
		pollInterval,
		propagationLimit,
		true,
		allConditions(
			recordHasPropagatedCheck(testNameServer, ch1.ResolvedFQDN, ch1.Key),
			recordHasPropagatedCheck(testNameServer, ch2.ResolvedFQDN, ch2.Key),
		)); err != nil {
		t.Errorf("error waiting for DNS record propagation: %v", err)
		return
	}

	t.Log("Cleaning up the second record only...")
	if err := solver.CleanUp(ch2); err != nil {
		t.Errorf("expected CleanUp to not error, but got: %v", err)
	}

	t.Log(
		"Waiting for propagation. Expecting first record to still exist, but " +
			"second record to be deleted...",
	)
	if err := wait.PollUntilContextTimeout(
		context.Background(),
		pollInterval,
		propagationLimit,
		true,
		allConditions(
			recordHasBeenDeletedCheck(testNameServer, ch2.ResolvedFQDN, ch2.Key),
			recordHasPropagatedCheck(testNameServer, ch1.ResolvedFQDN, ch1.Key),
		)); err != nil {
		t.Errorf("error waiting for DNS record propagation: %v", err)
		return
	}
}

func randomString() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func allConditions(c ...wait.ConditionWithContextFunc) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		for _, fn := range c {
			ok, err := fn(ctx)
			if err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	}
}

func recordHasPropagatedCheck(nameServer, fqdn, value string) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		return util.PreCheckDNS(ctx, fqdn, value, []string{nameServer}, true)
	}
}

func recordHasBeenDeletedCheck(nameServer, fqdn, value string) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		msg, err := util.DNSQuery(ctx, fqdn, dns.TypeTXT, []string{nameServer}, true)
		if err != nil {
			return false, err
		}
		if msg.Rcode == dns.RcodeNameError {
			return true, nil
		}
		if msg.Rcode != dns.RcodeSuccess {
			return false, fmt.Errorf("unexpected error from DNS server: %v", dns.RcodeToString[msg.Rcode])
		}
		for _, rr := range msg.Answer {
			txt, ok := rr.(*dns.TXT)
			if !ok {
				continue
			}
			for _, k := range txt.Txt {
				if k == value {
					return false, nil
				}
			}
		}
		return true, nil
	}
}
