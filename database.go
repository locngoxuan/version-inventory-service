package main

import (
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var db *bolt.DB
var bucketTransaction = []byte("transactions")
var bucketVersion = []byte("versions")

func initializeDatabase(base string) (err error) {
	err = os.MkdirAll(base, 0755)
	if err != nil {
		return err
	}
	dbFile := filepath.Join(base, "database")
	db, err = bolt.Open(dbFile, 0755, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		return err
	}

	err = db.Update(func(t *bolt.Tx) error {
		_, e := t.CreateBucketIfNotExists(bucketTransaction)
		if e != nil {
			return e
		}
		_, e = t.CreateBucketIfNotExists(bucketVersion)
		if e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func closeDb() {
	_ = db.Close()
}
