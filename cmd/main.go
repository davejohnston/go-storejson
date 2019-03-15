//go:generate go-bindata -nocompress -o ../internal/sql/sql.go -pkg sql ../internal/sql/...
package main

import (
	"time"

	"github.com/davejohnston/go-storejson/internal/data/multitable"
	"github.com/davejohnston/go-storejson/internal/storage"
	"github.com/davejohnston/go-storejson/internal/utils"
	_ "github.com/lib/pq"
	"github.com/puppetlabs/pd-proto/facet"
	"github.com/sirupsen/logrus"
)

const (
	facetMultipler = 4
	ingestThreads  = 30
)

func main() {
	db, err := utils.Connect()
	if err != nil {
		logrus.Fatalf("Unable to connect to database : %s", err)
	}

	multitable.DropSchema(db)
	err = multitable.CreateSchema(db)
	if err != nil {
		logrus.Fatalf("Unable to create schemas : %s", err)
	}

	facets := make(chan *facet.Instance, 10000)
	go func() {
		// Change to load multiples of the facets i*1742
		// 844 inaccessible
		// 20 containers
		logrus.Info("Loading Facets...")
		count := storage.Load(facets, "/tmp/tapes/mix-tape.json.gz", facetMultipler)
		logrus.Infof("Loaded %d Facets", count)
		close(facets)
	}()

	defer utils.TimeTrack(time.Now(), "PDP Ingest")
	count := multitable.Ingest(facets, ingestThreads)
	logrus.Infof("Processed %d facets", count)

}

func init() {
	logrus.SetLevel(logrus.InfoLevel)
}
