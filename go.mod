module lockbox.dev/grants

require (
	darlinggo.co/pan v0.2.0
	github.com/hashicorp/go-memdb v1.3.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/lib/pq v1.9.0
	github.com/rubenv/sql-migrate v0.0.0-20210215143335-f84234893558
	impractical.co/pqarrays v0.1.0
	yall.in v0.0.7
)

replace github.com/rubenv/sql-migrate => github.com/impractical/go-sql-migrate v0.0.1

go 1.16
