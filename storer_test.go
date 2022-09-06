package grants_test

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	uuid "github.com/hashicorp/go-uuid"
	"impractical.co/pqarrays"
	yall "yall.in"
	"yall.in/colour"

	"lockbox.dev/grants"
	"lockbox.dev/grants/storers/memory"
	"lockbox.dev/grants/storers/postgres"
)

type Factory interface {
	NewStorer(ctx context.Context) (grants.Storer, error)
	TeardownStorers() error
}

var factories []Factory

func uuidOrFail(t *testing.T) string {
	t.Helper()
	id, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatalf("Unexpected error generating ID: %s", err.Error())
	}
	return id
}

func TestMain(m *testing.M) {
	flag.Parse()

	// set up our test storers
	factories = append(factories, memory.Factory{})
	if os.Getenv(postgres.TestConnStringEnvVar) != "" {
		storerConn, err := sql.Open("postgres", os.Getenv(postgres.TestConnStringEnvVar))
		if err != nil {
			panic(err)
		}
		factories = append(factories, postgres.NewFactory(storerConn))
	}

	// run the tests
	result := m.Run()

	// tear down all the storers we created
	for _, factory := range factories {
		err := factory.TeardownStorers()
		if err != nil {
			log.Printf("Error cleaning up after %T: %+v\n", factory, err)
		}
	}

	// return the test result
	os.Exit(result)
}

func runTest(t *testing.T, testFunc func(*testing.T, grants.Storer, context.Context)) {
	logger := yall.New(colour.New(os.Stdout, yall.Debug))
	for _, factory := range factories {
		ctx := yall.InContext(context.Background(), logger)
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}
		t.Run(fmt.Sprintf("Storer=%T", storer), func(t *testing.T) {
			t.Parallel()
			testFunc(t, storer, ctx)
		})
	}
}

func TestCreateAndExchangeGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndExchangeGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		use := grants.GrantUse{Grant: grant.ID, IP: "8.8.8.8", Time: time.Now().Round(time.Millisecond)}
		resp, err := storer.ExchangeGrant(ctx, use)
		if err != nil {
			t.Errorf("Unexpected error exchanging grant in %T: %+v\n", storer, err)
		}
		expectation := grant
		expectation.Used = true
		expectation.UseIP = "8.8.8.8"
		expectation.UsedAt = use.Time
		if diff := cmp.Diff(expectation, resp); diff != "" {
			t.Errorf("Unexpected diff (-wanted, +got): %s", diff)
		}
	})
}

func TestCreateAndGetGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndExchangeGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		resp, err := storer.GetGrant(ctx, grant.ID)
		if err != nil {
			t.Errorf("Unexpected error retrieving grant from %T: %+v\n", storer, err)
		}
		expectation := grant
		if diff := cmp.Diff(expectation, resp); diff != "" {
			t.Errorf("Unexpected diff (-wanted, +got): %s", diff)
		}
	})
}

func TestCreateAndGetRootGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndExchangeGrant",
			AncestorIDs: []string{},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		resp, err := storer.GetGrant(ctx, grant.ID)
		if err != nil {
			t.Errorf("Unexpected error retrieving grant from %T: %+v\n", storer, err)
		}
		expectation := grant
		if diff := cmp.Diff(expectation, resp); diff != "" {
			t.Errorf("Unexpected diff (-wanted, +got): %s", diff)
		}
	})
}

func TestCreateAndGetGrantBySource(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndGetGrantBySource",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		resp, err := storer.GetGrantBySource(ctx, grant.SourceType, grant.SourceID)
		if err != nil {
			t.Errorf("Unexpected error retrieving grant from %T: %+v\n", storer, err)
		}
		expectation := grant
		if diff := cmp.Diff(expectation, resp); diff != "" {
			t.Errorf("Unexpected diff (-wanted, +got): %s", diff)
		}
	})
}

func TestCreateAndExchangeUsedGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndExchangeUsedGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		_, err = storer.ExchangeGrant(ctx, grants.GrantUse{Grant: grant.ID, IP: "1.2.3.4", Time: time.Now().Round(time.Millisecond)})
		if err != nil {
			t.Errorf("Unexpected error exchanging grant in %T: %+v\n", storer, err)
		}

		_, err = storer.ExchangeGrant(ctx, grants.GrantUse{Grant: grant.ID, IP: "5.6.7.8", Time: time.Now().Round(time.Millisecond)})
		if !errors.Is(err, grants.ErrGrantAlreadyUsed) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantAlreadyUsed, storer, err)
		}
	})
}

func TestCreateAndExchangeRevokedGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndExchangeRevokedGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		_, err = storer.RevokeGrant(ctx, grant.ID)
		if err != nil {
			t.Errorf("Unexpected error exchanging grant in %T: %+v\n", storer, err)
		}

		_, err = storer.ExchangeGrant(ctx, grants.GrantUse{Grant: grant.ID, IP: "5.6.7.8", Time: time.Now().Round(time.Millisecond)})
		if !errors.Is(err, grants.ErrGrantRevoked) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantAlreadyUsed, storer, err)
		}
	})
}

func TestExchangeNonExistentGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		_, err := storer.ExchangeGrant(ctx, grants.GrantUse{
			Grant: uuidOrFail(t),
			IP:    "8.8.8.8",
			Time:  time.Now().Round(time.Millisecond),
		})
		if !errors.Is(err, grants.ErrGrantNotFound) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantNotFound, storer, err)
		}
	})
}

func TestGetNonExistentGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		_, err := storer.GetGrant(ctx, uuidOrFail(t))
		if !errors.Is(err, grants.ErrGrantNotFound) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantNotFound, storer, err)
		}
	})
}

func TestGetNonExistentGrantBySource(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		_, err := storer.GetGrantBySource(ctx, "test", "non-existent-grant")
		if !errors.Is(err, grants.ErrGrantNotFound) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantNotFound, storer, err)
		}
	})
}

func TestCreateDuplicateGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateDuplicateGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		grant.SourceID += "!"

		err = storer.CreateGrant(ctx, grant)
		if !errors.Is(err, grants.ErrGrantAlreadyExists) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantAlreadyExists, storer, err)
		}
	})
}

func TestCreateDuplicateSourceGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateDuplicateSourceGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		grant.ID = uuidOrFail(t)

		err = storer.CreateGrant(ctx, grant)
		if !errors.Is(err, grants.ErrGrantSourceAlreadyUsed) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantSourceAlreadyUsed, storer, err)
		}
	})
}

func TestCreateAndRevokeGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndRevokeGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		resp, err := storer.RevokeGrant(ctx, grant.ID)
		if err != nil {
			t.Errorf("Unexpected error exchanging grant in %T: %+v\n", storer, err)
		}
		expectation := grant
		expectation.Revoked = true
		if diff := cmp.Diff(expectation, resp); diff != "" {
			t.Errorf("Unexpected diff (-wanted, +got): %s", diff)
		}
	})
}

func TestCreateAndRevokeUsedGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:          uuidOrFail(t),
			SourceType:  "manual",
			SourceID:    "TestCreateAndRevokeUsedGrant",
			AncestorIDs: pqarrays.StringArray{uuidOrFail(t), uuidOrFail(t)},
			UsedAt:      time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:      pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:   "tester",
			AccountID:   "test123",
			ClientID:    "testrunner",
			CreateIP:    "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		_, err = storer.ExchangeGrant(ctx, grants.GrantUse{Grant: grant.ID, IP: "1.2.3.4", Time: time.Now().Round(time.Millisecond)})
		if err != nil {
			t.Errorf("Unexpected error exchanging grant in %T: %+v\n", storer, err)
		}

		_, err = storer.RevokeGrant(ctx, grant.ID)
		if !errors.Is(err, grants.ErrGrantAlreadyUsed) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantAlreadyUsed, storer, err)
		}
	})
}

func TestRevokeNonExistentGrant(t *testing.T) {
	t.Parallel()

	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		_, err := storer.RevokeGrant(ctx, uuidOrFail(t))
		if !errors.Is(err, grants.ErrGrantNotFound) {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantNotFound, storer, err)
		}
	})
}
