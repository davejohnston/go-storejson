// Code generated by go-bindata.
// sources:
// ../internal/sql/edgeTbl.sql
// ../internal/sql/resourceTbl.sql
// ../internal/sql/sql.go
// DO NOT EDIT!

package sql

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)
type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _InternalSqlEdgetblSql = []byte(`CREATE TABLE __table__ (
  parent_id text NOT NULL,
  child_id text NOT NULL,
  type text,
  PRIMARY KEY (parent_id, child_id),
  FOREIGN KEY (parent_id) REFERENCES nm_resource(id) ON UPDATE CASCADE,
  FOREIGN KEY (child_id) REFERENCES nm_vulnerability(id) ON UPDATE CASCADE
);
ALTER TABLE edges SET UNLOGGED`)

func InternalSqlEdgetblSqlBytes() ([]byte, error) {
	return _InternalSqlEdgetblSql, nil
}

func InternalSqlEdgetblSql() (*asset, error) {
	bytes, err := InternalSqlEdgetblSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "../internal/sql/edgeTbl.sql", size: 308, mode: os.FileMode(420), modTime: time.Unix(1552655138, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _InternalSqlResourcetblSql = []byte(`CREATE TABLE __table__ (
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
$$;`)

func InternalSqlResourcetblSqlBytes() ([]byte, error) {
	return _InternalSqlResourcetblSql, nil
}

func InternalSqlResourcetblSql() (*asset, error) {
	bytes, err := InternalSqlResourcetblSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "../internal/sql/resourceTbl.sql", size: 840, mode: os.FileMode(420), modTime: time.Unix(1552649139, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _InternalSqlSqlGo = []byte(``)

func InternalSqlSqlGoBytes() ([]byte, error) {
	return _InternalSqlSqlGo, nil
}

func InternalSqlSqlGo() (*asset, error) {
	bytes, err := InternalSqlSqlGoBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "../internal/sql/sql.go", size: 0, mode: os.FileMode(420), modTime: time.Unix(1552655143, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"../internal/sql/edgeTbl.sql": InternalSqlEdgetblSql,
	"../internal/sql/resourceTbl.sql": InternalSqlResourcetblSql,
	"../internal/sql/sql.go": InternalSqlSqlGo,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"..": &bintree{nil, map[string]*bintree{
		"internal": &bintree{nil, map[string]*bintree{
			"sql": &bintree{nil, map[string]*bintree{
				"edgeTbl.sql": &bintree{InternalSqlEdgetblSql, map[string]*bintree{}},
				"resourceTbl.sql": &bintree{InternalSqlResourcetblSql, map[string]*bintree{}},
				"sql.go": &bintree{InternalSqlSqlGo, map[string]*bintree{}},
			}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

