package pdp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/davejohnston/go-storejson/internal/utils"
	"github.com/jmoiron/sqlx"
	"github.com/puppetlabs/cloud-discovery/pdpgo/pkg/model"
	"github.com/puppetlabs/pd-proto/facet"
	"github.com/sirupsen/logrus"
)

var (
	createTable = `
CREATE TABLE resource (
	id	TEXT,
	parent_id TEXT,
	type TEXT,
	props jsonb DEFAULT '{}'::jsonb,
	created_at TIMESTAMP with time zone,
	updated_at TIMESTAMP with time zone,
	PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_updated_at ON resource (updated_at);
CREATE INDEX IF NOT EXISTS idx_resource_parent ON resource (parent_id);
CREATE INDEX IF NOT EXISTS idx_resource_type ON resource (type);
CREATE INDEX IF NOT EXISTS idx_resource_props_gin ON resource USING gin (props);
CREATE OR REPLACE FUNCTION jsonb_shallow_merge(a jsonb, b jsonb)
returns jsonb language sql as $$
	select
		jsonb_object_agg(
			coalesce(ka, kb),
			case
				when va isnull then vb
				when vb isnull then va
					else va || vb
			end
		)
	from jsonb_each(a) e1(ka, va)
	full join jsonb_each(b) e2(kb, vb) on ka = kb
$$;
`
)

func CreateSchema(db *sqlx.DB) error {
	logrus.Info("Creating Table [resource]")
	return utils.Exec(db, createTable)

}

func DropSchema(db *sqlx.DB) {
	logrus.Info("Dropping Table [resource]")
	utils.Exec(db, "DROP TABLE IF EXISTS resource;")
}

func Ingest(db *sqlx.DB, facets chan *facet.Instance) uint64 {

	var count uint64
	for facet := range facets {

		if facet.ID != "" {
			if facet.SchemaID.Namespace == "system" && facet.SchemaID.Name == "childIDs" {
				// Do nothing for test purposes
			} else {
				key, values, err := model.FacetToInternal(facet)
				if err != nil {
					logrus.Warnf("Error converting facet: %#v", err)
					break
				}

				if key.Parent != "" && key.Parent != key.ID {
					// this catches strange behavior where fake-data (and possibly some providers)
					// send resources without the full path inscribed in their ID
					if !strings.Contains(key.ID, "|") {
						key.ID = strings.Join([]string{key.Parent, key.ID}, "|")
					}
				}

				newResource := map[model.ResourceKey]model.Resource{*key: values}
				store(db, newResource)
			}
			count++
		}
	}
	return count
}

func store(db *sqlx.DB, resources map[model.ResourceKey]model.Resource) error {
	currentTime := time.Now()

	var keys []string
	idToKey := make(map[string]model.ResourceKey)
	for k := range resources {
		keys = append(keys, k.ID)
		idToKey[k.ID] = k
	}

	sort.Strings(keys)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		// Rollback if a commit fails
		tx.Rollback()
	}()

	var nodesArray []interface{}

	for _, resourceID := range keys {
		resourceKey := idToKey[resourceID]
		resource := resources[resourceKey]

		facetMap := make(map[string]interface{})
		facetMapJSON, _ := json.Marshal(make(map[string]interface{}))

		// create a map of facets.
		var rType = ""
		for schemaID, resourceValue := range resource {
			facetID := strings.Join([]string{schemaID.Namespace, schemaID.Name}, "_")

			facetMap[facetID] = resourceValue

			facetMapJSON, err = json.Marshal(facetMap)
			if err != nil {
				return err
			}

			// This will get overwritten each time, but should resovle to the
			// correct type regardless
			rType = getType(schemaID)
		}

		// we create the base node with type 'resource'.
		nodesArray = append(nodesArray,
			resourceKey.ID,
			resourceKey.Parent,
			rType,
			facetMapJSON,
			currentTime,
			currentTime)
	}

	if len(nodesArray) > 0 {

		valueStatement := "("
		for nodesIndex := range nodesArray {
			// 8 is the number of properties for nodes
			if (nodesIndex+1)%6 == 0 && nodesIndex != 0 {
				if nodesIndex == len(nodesArray)-1 {
					valueStatement += fmt.Sprintf("$%v)", nodesIndex+1)
				} else {
					valueStatement += fmt.Sprintf("$%v),(", nodesIndex+1)
				}
			} else {
				valueStatement += fmt.Sprintf("$%v,", nodesIndex+1)
			}
		}
		q := fmt.Sprintf(`INSERT INTO resource
					(id, parent_id, type, props, created_at, updated_at)
				VALUES
					%s
				ON CONFLICT (id)
				DO UPDATE SET
					props=jsonb_shallow_merge((resource.props)::jsonb, (EXCLUDED.props)::jsonb), updated_at=EXCLUDED.updated_at`, valueStatement)
		_, err = tx.Exec(q, nodesArray...)
		if err != nil {
			logrus.Errorf("Failed to create Resource batch: %#v", err)
			return err
		}
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		logrus.Errorf("Transaction Commit Failed: %#v", err)
		return err
	}

	return err
}

func getType(schemaID model.SchemaID) string {

	if schemaID.Namespace == "pd" {
		switch schemaID.Name {
		case "host", "vm", "cloudResource", "computeHost":
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
