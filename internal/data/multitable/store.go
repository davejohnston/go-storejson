package multitable

import (
	"encoding/json"
	"time"

	"github.com/davejohnston/go-storejson/internal/sql"
	"github.com/davejohnston/go-storejson/internal/utils"
	"github.com/jmoiron/sqlx"
	"github.com/puppetlabs/pd-proto/facet"
	"github.com/sirupsen/logrus"

	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	resourceTable = string(sql.MustAsset("../internal/sql/resourceTbl.sql"))
	edgeTable     = string(sql.MustAsset("../internal/sql/edgeTbl.sql"))

	primary_tables = map[string]string{
		"resource":         resourceTable, // The original table
		"pd_container":     resourceTable,
		"pd_group":         resourceTable,
		"pd_host":          resourceTable,
		"pd_package":       resourceTable,
		"pd_service":       resourceTable,
		"pd_user":          resourceTable,
		"pd_vulnerability": resourceTable,
		"pd_resource":      resourceTable,
		"nm_vulnerability": resourceTable,
		"nm_resource":      resourceTable,
	}

	secondary_tables = map[string]string{
		"edges": edgeTable,
	}
)

func CreateSchema(db *sqlx.DB) error {
	err := createTables(db, primary_tables)
	if err != nil {
		return err
	}

	err = createTables(db, secondary_tables)
	if err != nil {
		return err
	}

	return nil
}

func DropSchema(db *sqlx.DB) {
	dropTables(db, secondary_tables)
	dropTables(db, primary_tables)
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
			tableName := "pd_resource"
			if f.SchemaID.Namespace == "pd" {
				tableName = rType
			}

			// TODO really should convert this to a Tx to store many records in one
			// tx
			save(db, tableName, facetID, parentPath, rType, props, currentTime)
			save(db, "resource", facetID, parentPath, rType, props, currentTime)

			// Create special link for vulnerability
			if f.SchemaID.Name == "vulnerability" {
				// Save Vulnerability Portion
				saveVR(db, facetID, parentPath, rType, props, currentTime)
				continue
			}
		}
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

func createTables(db *sqlx.DB, tables map[string]string) error {
	for tbl, query := range tables {
		logrus.Infof("Creating Table [%s]", tbl)
		err := createTable(db, tbl, query)
		if err != nil {
			return err
		}
	}

	return nil
}

func createTable(db *sqlx.DB, tableName string, query string) error {
	q := strings.Replace(query, "__table__", tableName, -1)
	return utils.Exec(db, q)
}

func dropTables(db *sqlx.DB, tables map[string]string) {
	for tbl, _ := range tables {
		logrus.Infof("Dropping Table [%s]", tbl)
		utils.Exec(db, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tbl))
	}
}

func saveVR(db *sqlx.DB, id string, parentID string, rType string, props []byte, currentTime time.Time) {
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

	tbl = "edges"
	q = fmt.Sprintf(`INSERT INTO %s (parent_id,child_id,type)
									VALUES ($1,$2, $3);`, tbl)
	_, err = db.Exec(q, id, shortId, rType)
	if err != nil {
		logrus.Errorf("Failed to create Resource batch: %#v", err)
		return
	}

}
