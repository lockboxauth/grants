package grants_test

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	uuid "github.com/hashicorp/go-uuid"
	"impractical.co/auth/grants"
	"impractical.co/auth/grants/storers/memory"
	"impractical.co/auth/grants/storers/postgres"
	"impractical.co/pqarrays"
	yall "yall.in"
	"yall.in/colour"
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

func compareGrants(grant1, grant2 grants.Grant) (success bool, field string, val1, val2 interface{}) {
	if grant1.ID != grant2.ID {
		return false, "ID", grant1.ID, grant2.ID
	}
	if grant1.SourceType != grant2.SourceType {
		return false, "SourceType", grant1.SourceType, grant2.SourceType
	}
	if grant1.SourceID != grant2.SourceID {
		return false, "SourceID", grant1.SourceID, grant2.SourceID
	}
	if !grant1.CreatedAt.Equal(grant2.CreatedAt) {
		return false, "CreatedAt", grant1.CreatedAt, grant2.CreatedAt
	}
	if len(grant1.Scopes) != len(grant2.Scopes) {
		return false, "Scopes", grant1.Scopes, grant2.Scopes
	}
	for pos, scope := range grant1.Scopes {
		if grant2.Scopes[pos] != scope {
			return false, "Scopes#" + strconv.Itoa(pos), grant1.Scopes, grant2.Scopes
		}
	}
	if grant1.ProfileID != grant2.ProfileID {
		return false, "ProfileID", grant1.ProfileID, grant2.ProfileID
	}
	if grant1.ClientID != grant2.ClientID {
		return false, "ClientID", grant1.ClientID, grant2.ClientID
	}
	if grant1.CreateIP != grant2.CreateIP {
		return false, "CreateIP", grant1.CreateIP, grant2.CreateIP
	}
	if grant2.UseIP != grant2.UseIP {
		return false, "UseIP", grant1.UseIP, grant2.UseIP
	}
	if grant1.Used != grant2.Used {
		return false, "Used", grant1.Used, grant2.Used
	}
	if !grant1.UsedAt.Equal(grant2.UsedAt) {
		return false, "UsedAt", grant1.UsedAt, grant2.UsedAt
	}
	return true, "", nil, nil
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

func runTest(t *testing.T, f func(*testing.T, grants.Storer, context.Context)) {
	t.Parallel()
	logger := yall.New(colour.New(os.Stdout, yall.Debug))
	for _, factory := range factories {
		ctx := yall.InContext(context.Background(), logger)
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}
		t.Run(fmt.Sprintf("Storer=%T", storer), func(t *testing.T) {
			t.Parallel()
			f(t, storer, ctx)
		})
	}
}

func TestCreateAndExchangeGrant(t *testing.T) {
	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:         uuidOrFail(t),
			SourceType: "manual",
			SourceID:   "TestCreateAndExchangeGrant",
			UsedAt:     time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			CreateIP:   "192.168.1.2",
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
		ok, field, expected, result := compareGrants(expectation, resp)
		if !ok {
			t.Errorf("Expected %s to be %v in %T, got %v\n", field, expected, storer, result)
		}
	})
}

func TestCreateAndExchangeUsedGrant(t *testing.T) {
	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:         uuidOrFail(t),
			SourceType: "manual",
			SourceID:   "TestCreateAndExchangeUsedGrant",
			UsedAt:     time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			CreateIP:   "192.168.1.2",
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
		if err != grants.ErrGrantAlreadyUsed {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantAlreadyUsed, storer, err)
		}
	})
}

func TestExchangeNonExistentGrant(t *testing.T) {
	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		_, err := storer.ExchangeGrant(ctx, grants.GrantUse{
			Grant: uuidOrFail(t),
			IP:    "8.8.8.8",
			Time:  time.Now().Round(time.Millisecond),
		})
		if err != grants.ErrGrantNotFound {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantNotFound, storer, err)
		}
	})
}

func TestCreateDuplicateGrant(t *testing.T) {
	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:         uuidOrFail(t),
			SourceType: "manual",
			SourceID:   "TestCreateDuplicateGrant",
			UsedAt:     time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			CreateIP:   "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		grant.SourceID += "!"

		err = storer.CreateGrant(ctx, grant)
		if err != grants.ErrGrantAlreadyExists {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantAlreadyExists, storer, err)
		}
	})
}

func TestCreateDuplicateSourceGrant(t *testing.T) {
	runTest(t, func(t *testing.T, storer grants.Storer, ctx context.Context) {
		grant := grants.Grant{
			ID:         uuidOrFail(t),
			SourceType: "manual",
			SourceID:   "TestCreateDuplicateSourceGrant",
			UsedAt:     time.Now().Add(time.Hour).Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			CreateIP:   "192.168.1.2",
		}
		err := storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		grant.ID = uuidOrFail(t)

		err = storer.CreateGrant(ctx, grant)
		if err != grants.ErrGrantSourceAlreadyUsed {
			t.Errorf("Expected error to be %v, %T returned %v\n", grants.ErrGrantSourceAlreadyUsed, storer, err)
		}
	})
}
