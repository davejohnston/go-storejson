CREATE TABLE __table__ (
  parent_id text NOT NULL,
  child_id text NOT NULL,
  type text,
  PRIMARY KEY (parent_id, child_id),
  FOREIGN KEY (parent_id) REFERENCES nm_resource(id) ON UPDATE CASCADE,
  FOREIGN KEY (child_id) REFERENCES nm_vulnerability(id) ON UPDATE CASCADE
);
ALTER TABLE edges SET UNLOGGED