package normalize

import (
	"encoding/json"
	"time"

	"github.com/davejohnston/go-storejson/internal/utils"
	"github.com/jmoiron/sqlx"
	"github.com/puppetlabs/pd-proto/facet"
	"github.com/sirupsen/logrus"

	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	table = `
CREATE TABLE __table__ (
	id	TEXT,
	parent_id TEXT,
	type TEXT,
	props jsonb DEFAULT '{}'::jsonb,
	created_at TIMESTAMP with time zone,
	updated_at TIMESTAMP with time zone,
	PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx___table___updated_at ON __table__ (updated_at);
CREATE INDEX IF NOT EXISTS idx___table___parent ON __table__ (parent_id);
CREATE INDEX IF NOT EXISTS idx___table___type ON __table__ (type);
CREATE INDEX IF NOT EXISTS idx___table___props_gin ON __table__ USING gin (props);
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

	table2 = `
CREATE TABLE instances_of (
  parent_id text NOT NULL,
  child_id text NOT NULL,
  type text,
  PRIMARY KEY (parent_id, child_id),
  FOREIGN KEY (parent_id) REFERENCES nm_resource(id) ON UPDATE CASCADE,
  FOREIGN KEY (child_id) REFERENCES nm_vulnerability(id) ON UPDATE CASCADE
);
ALTER TABLE instances_of SET UNLOGGED
`
)

var (
	tables = []string{"nm_vulnerability", "nm_resource"}
)

func CreateSchema(db *sqlx.DB) error {
	for _, tbl := range tables {
		logrus.Infof("Creating Table [%s]", tbl)
		err := createTable(db, tbl)
		if err != nil {
			return err
		}
	}

	err := utils.Exec(db, table2)
	if err != nil {
		return err
	}
	return nil
}

func DropSchema(db *sqlx.DB) {
	for _, tbl := range tables {
		logrus.Infof("Dropping Table [%s]", tbl)
		utils.Exec(db, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tbl))
	}
	utils.Exec(db, fmt.Sprintf("DROP TABLE IF EXISTS %s;", "instances_of"))
}

// Ingest starts a number of go routines specified by ingestThreads
// each thread takes care of storing the data
// The function  blocks until all routines return.
func Ingest(facets chan *facet.Instance, ingestThreads int) uint64 {
	var count uint64

	wg := sync.WaitGroup{}
	for i := 0; i < ingestThreads; i++ {
		wg.Add(1)
		go func(id int) {
			logrus.Debugf("Staring worker %d", id)
			result := process(facets)
			logrus.Debugf("Worker %d returned %d facets", id, result)
			atomic.AddUint64(&count, result)
			wg.Done()
		}(i)
	}

	wg.Wait()

	return count
}

func process(facets chan *facet.Instance) uint64 {

	db, err := utils.Connect()
	if err != nil {
		logrus.Error(err)
		return 0
	}

	var count = 0
	var index = 0
	buffer := make([]*facet.Instance, 0)
	for f := range facets {
		//if f.SchemaID.Name == "host" || f.SchemaID.Name == "vulnerability" {
		if f.SchemaID.Name == "vulnerability" {
			buffer = append(buffer, f)
			index++
			if index == 100 {
				// Process Batch
				store(db, buffer)
				buffer = make([]*facet.Instance, 0)
				count += index
				index = 0
			}
		}
	}

	logrus.Infof("Processing remaining %d facets", len(buffer))
	count += index
	return uint64(count)
}

func store(db *sqlx.DB, facets []*facet.Instance) {
	currentTime := time.Now()
	for _, f := range facets {

		if f.SchemaID.Namespace == "system" && f.SchemaID.Name == "childIDs" {
			// Do nothing for test purposes
		} else {

			facetID := f.ID
			parentPath := strings.Join(f.ParentPath, "|")
			if !strings.Contains(f.ID, "|") {
				facetID = strings.Join([]string{parentPath, f.ID}, "|")
			}

			propsMap := make(map[string]interface{})
			json.Unmarshal([]byte(f.Attributes), &propsMap)

			attrMap := map[string]interface{}{strings.Join([]string{f.SchemaID.Namespace, f.SchemaID.Name}, "_"): propsMap}
			props, err := json.Marshal(attrMap)
			if err != nil {
				logrus.Error(err)
				continue
			}

			rType := utils.GetType(f.SchemaID)
			tableName := "nm_resource"
			if f.SchemaID.Name == "vulnerability" {
				// Save Vulnerability Portion
				saveVR(db, facetID, parentPath, rType, props, currentTime)
				continue
			}

			// TODO really should convert this to a Tx to store many records in one
			// tx
			save(db, tableName, facetID, parentPath, rType, props, currentTime)
		}
	}
}

func saveVR(db *sqlx.DB, id string, parentID string, rType string, props []byte, currentTime time.Time) {

	// Need to split the VR into 2 facets
	//
	// extract the ID and use that

	/*
			| VR ID | Props |
			|    1  |  ...  |
			-----------------
			|    2  | ..... |
			-----------------


			VR_Link
			|  ID | Parent_Path | Props |
			-----------------------------
			|     |             |  vr_id :  host_id |
		 	-----------------------------

	*/
	rType = "nm_vulnerability"
	parts := strings.Split(id, "|")
	shortId := parts[len(parts)-1]

	tbl := "nm_vulnerability"

	q := fmt.Sprintf(`INSERT INTO %s (id,type,props,created_at,updated_at)
									VALUES ($1,$2,$3,$4,$5)
									ON CONFLICT (id) DO UPDATE SET
									props=jsonb_shallow_merge((%s.props)::jsonb, (EXCLUDED.props)::jsonb), updated_at=EXCLUDED.updated_at`, tbl, tbl)
	_, err := db.Exec(q, shortId, rType, props, currentTime, currentTime)
	if err != nil {
		logrus.Errorf("Failed to create Resource batch: %#v", err)
		return
	}

	tbl = "nm_resource"
	q = fmt.Sprintf(`INSERT INTO %s (id,parent_id,type,created_at,updated_at)
									VALUES ($1,$2,$3,$4,$5)
									ON CONFLICT (id) DO UPDATE SET
									props=jsonb_shallow_merge((%s.props)::jsonb, (EXCLUDED.props)::jsonb), updated_at=EXCLUDED.updated_at`, tbl, tbl)
	_, err = db.Exec(q, id, parentID, rType, currentTime, currentTime)
	if err != nil {
		logrus.Errorf("Failed to create Resource batch: %#v", err)
		return
	}

	tbl = "instances_of"
	q = fmt.Sprintf(`INSERT INTO %s (parent_id,child_id,type)
									VALUES ($1,$2, $3);`, tbl)
	_, err = db.Exec(q, id, shortId, rType)
	if err != nil {
		logrus.Errorf("Failed to create Resource batch: %#v", err)
		return
	}

}

func save(db *sqlx.DB, table string, id string, parentID string, rType string, props []byte, currentTime time.Time) {

	q := fmt.Sprintf(`INSERT INTO %s (id,parent_id,type,props,created_at,updated_at)
									VALUES ($1,$2,$3,$4,$5,$6)
									ON CONFLICT (id) DO UPDATE SET
									props=jsonb_shallow_merge((%s.props)::jsonb, (EXCLUDED.props)::jsonb), updated_at=EXCLUDED.updated_at`, table, table)
	_, err := db.Exec(q, id, parentID, rType, props, currentTime, currentTime)
	if err != nil {
		logrus.Errorf("Failed to create Resource batch: %#v", err)
		return
	}
}

func createTable(db *sqlx.DB, tableName string) error {
	query := strings.Replace(table, "__table__", tableName, -1)
	return utils.Exec(db, query)
}
