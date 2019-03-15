package utils

import (
	"fmt"

	"strings"

	"time"

	"github.com/jmoiron/sqlx"
	"github.com/puppetlabs/pd-proto/facet"
	"github.com/sirupsen/logrus"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "postgres"
)

// Exec a postgres query, returning an error on failure
func Exec(db *sqlx.DB, statement string) error {
	if db != nil {
		r, err := db.Exec(statement)
		if err != nil {
			return err
		}
		rows, _ := r.RowsAffected()
		logrus.Debugf("Affected Rows %d", rows)
	} else {
		return fmt.Errorf("Database connection is nil")
	}

	return nil
}

// Connect to postgres and return the DB connection
func Connect() (*sqlx.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	logrus.Debugf("Successfully connected to database %s", psqlInfo)
	return db, nil
}

// GetType returns the resource type given a facet schema ID e.g.
// a host facet is a host resource, but a computeHost facet is also
// a host
func GetType(schemaID *facet.SchemaID) string {

	if schemaID.Namespace == "pd" {
		switch schemaID.Name {
		case "host", "vm", "cloudResource", "computeHost", "discovery":
			return fmt.Sprintf("%s_host", schemaID.Namespace)
		}
	} else {
		// some special cases.  Longer term we should update the facets, to
		// provide this info.
		switch schemaID.Name {
		case "computeInstance":
			return "pd_host"
		}
	}

	return strings.Join([]string{schemaID.Namespace, schemaID.Name}, "_")
}

// TimeTrack logs the time an operation takes to run
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	logrus.Infof("%s took %s to complete ingest", name, elapsed)
}
