package storage

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"os"

	"fmt"

	"github.com/puppetlabs/pd-proto/facet"
	"github.com/sirupsen/logrus"
)

// Load streams the facets from the file filename into the facets channel.
func Load(facets chan *facet.Instance, filename string, multipler int) int {
	count := 0

	for i := 1; i < multipler+1; i++ {
		id := fmt.Sprintf("00%d", i)
		err, l := streamFacets(id, filename, facets)
		if err != nil {
			logrus.Error(err)
		}
		logrus.Debugf("%s Loaded %d Facets", id, l)
		count += l
	}
	return count
}

func streamFacets(id string, filename string, facets chan *facet.Instance) (error, int) {
	var count = 0

	f, err := os.Open(filename)
	if err != nil {
		return err, 0
	}
	defer f.Close()

	reader, err := gzip.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	dec := json.NewDecoder(reader)

	//read open bracket
	_, err = dec.Token()
	if err != nil {
		return err, 0
	}

	// while the array contains values
	for dec.More() {
		var m facet.Instance
		// decode an array value (Message)
		err := dec.Decode(&m)
		if err != nil {
			return err, 0
		}

		// Modify the parent_path
		if len(m.ParentPath) >= 1 {
			parent := fmt.Sprintf("%s_%s", id, m.ParentPath[0])
			m.ParentPath[0] = parent
		}

		facets <- &m
		count++
	}

	// read closing bracket
	_, err = dec.Token()
	if err != nil {
		return err, 0
	}

	return nil, count
}
