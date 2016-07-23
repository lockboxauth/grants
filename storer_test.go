package grants

import (
	"flag"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"code.impractical.co/pqarrays"
	"github.com/pborman/uuid"

	"golang.org/x/net/context"
)

type StorerFactory interface {
	NewStorer(ctx context.Context) (Storer, error)
	TeardownStorers() error
}

var storerFactories []StorerFactory

func compareGrants(grant1, grant2 Grant) (success bool, field string, val1, val2 interface{}) {
	if !uuid.Equal(grant1.ID, grant2.ID) {
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
	if grant1.IP != grant2.IP {
		return false, "IP", grant1.IP, grant2.IP
	}
	if grant1.Used != grant2.Used {
		return false, "Used", grant1.Used, grant2.Used
	}
	return true, "", nil, nil
}

func TestMain(m *testing.M) {
	flag.Parse()
	result := m.Run()
	for _, factory := range storerFactories {
		err := factory.TeardownStorers()
		if err != nil {
			log.Printf("Error cleaning up after %T: %+v\n", factory, err)
		}
	}
	os.Exit(result)
}

func TestCreateAndExchangeGrant(t *testing.T) {
	t.Parallel()
	for _, factory := range storerFactories {
		ctx := context.Background()
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}

		grant := Grant{
			ID:         uuid.NewRandom(),
			SourceType: "manual",
			SourceID:   "TestCreateAndExchangeGrant",
			CreatedAt:  time.Now().Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			IP:         "192.168.1.2",
		}
		err = storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		resp, err := storer.ExchangeGrant(ctx, grant.ID)
		if err != nil {
			t.Errorf("Unexpected error exchanging grant in %T: %+v\n", storer, err)
		}
		expectation := grant
		expectation.Used = true
		ok, field, expected, result := compareGrants(expectation, resp)
		if !ok {
			t.Errorf("Expected %s to be %v in %T, got %v\n", field, expected, storer, result)
		}
	}
}

func TestCreateAndExchangeUsedGrant(t *testing.T) {
	t.Parallel()
	for _, factory := range storerFactories {
		ctx := context.Background()
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}

		grant := Grant{
			ID:         uuid.NewRandom(),
			SourceType: "manual",
			SourceID:   "TestCreateAndExchangeUsedGrant",
			CreatedAt:  time.Now().Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			IP:         "192.168.1.2",
		}
		err = storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		_, err = storer.ExchangeGrant(ctx, grant.ID)
		if err != nil {
			t.Errorf("Unexpected error exchanging grant in %T: %+v\n", storer, err)
		}

		_, err = storer.ExchangeGrant(ctx, grant.ID)
		if err != ErrGrantAlreadyUsed {
			t.Errorf("Expected error to be %v, %T returned %v\n", ErrGrantAlreadyUsed, storer, err)
		}
	}
}

func TestExchangeNonExistentGrant(t *testing.T) {
	t.Parallel()
	for _, factory := range storerFactories {
		ctx := context.Background()
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}

		_, err = storer.ExchangeGrant(ctx, uuid.NewRandom())
		if err != ErrGrantNotFound {
			t.Errorf("Expected error to be %v, %T returned %v\n", ErrGrantNotFound, storer, err)
		}
	}
}

func TestCreateDuplicateGrant(t *testing.T) {
	t.Parallel()
	for _, factory := range storerFactories {
		ctx := context.Background()
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}

		grant := Grant{
			ID:         uuid.NewRandom(),
			SourceType: "manual",
			SourceID:   "TestCreateDuplicateGrant",
			CreatedAt:  time.Now().Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			IP:         "192.168.1.2",
		}
		err = storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		grant.SourceID += "!"

		err = storer.CreateGrant(ctx, grant)
		if err != ErrGrantAlreadyExists {
			t.Errorf("Expected error to be %v, %T returned %v\n", ErrGrantAlreadyExists, storer, err)
		}
	}
}

func TestCreateDuplicateSourceGrant(t *testing.T) {
	t.Parallel()
	for _, factory := range storerFactories {
		ctx := context.Background()
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}

		grant := Grant{
			ID:         uuid.NewRandom(),
			SourceType: "manual",
			SourceID:   "TestCreateDuplicateSourceGrant",
			CreatedAt:  time.Now().Round(time.Millisecond),
			Scopes:     pqarrays.StringArray{"https://scopes.impractical.co/test", "https://scopes.impractical.co/other/test"},
			ProfileID:  "tester",
			ClientID:   "testrunner",
			IP:         "192.168.1.2",
		}
		err = storer.CreateGrant(ctx, grant)
		if err != nil {
			t.Errorf("Unexpected error creating grant in %T: %+v\n", storer, err)
		}

		grant.ID = uuid.NewRandom()

		err = storer.CreateGrant(ctx, grant)
		if err != ErrGrantSourceAlreadyUsed {
			t.Errorf("Expected error to be %v, %T returned %v\n", ErrGrantSourceAlreadyUsed, storer, err)
		}
	}
}
