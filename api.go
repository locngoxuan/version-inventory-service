package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/narqo/go-badge"
	"go.etcd.io/bbolt"
)

type (
	PrepareRequest struct {
		RepoId        string `json:"repo_id,omitempty"`
		ReleaseAction string `json:"action,omitempty"`
		Version       string `json:"version,omitempty"`
	}

	CommitRequest struct {
		TxId string `json:"tx_id,omitempty"`
	}
)

func prepareVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request PrepareRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.Errorw("failed to decode request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	action := "latest"
	if v := strings.TrimSpace(request.ReleaseAction); v != "" {
		action = v
	}
	if strings.TrimSpace(request.RepoId) == "" ||
		strings.TrimSpace(request.Version) == "" {
		logger.Error("repo_id or version is missing")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	txId := NewObjectID().Hex()
	err = db.Update(func(t *bbolt.Tx) error {
		b := t.Bucket(bucketTransaction)
		m := map[string]string{
			"repo":    strings.TrimSpace(request.RepoId),
			"action":  action,
			"version": strings.TrimSpace(request.Version),
		}
		bs, err := json.Marshal(m)
		if err != nil {
			logger.Errorw("failed to marshal data before put to bucket", "err", err, "data", m)
			return err
		}
		b.Put([]byte(txId), bs)
		return nil
	})
	if err != nil {
		logger.Errorw("failed to put data to database", "err", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(txId))
}

func commitVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request CommitRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.Errorw("failed to decode request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	txId := strings.TrimSpace(request.TxId)
	if txId == "" {
		logger.Error("tx_id is missing")
		http.Error(w, "missing txId", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin(true)
	if err != nil {
		logger.Errorw("failed to start db transaction", "err", err, "tx", txId)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	txBucket := tx.Bucket(bucketTransaction)
	txVal := txBucket.Get([]byte(txId))
	if txVal == nil || len(txVal) == 0 {
		logger.Errorw("transaction does not exist or was already committed", "tx", txId)
		http.Error(w, "transaction does not exist", http.StatusNotFound)
		return
	}
	var txM map[string]string
	err = json.Unmarshal(txVal, &txM)
	if err != nil {
		logger.Errorw("failed to unmarshal transaction content", "data", string(txVal),
			"err", err, "tx", txId)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	repoId := txM["repo"]
	action := txM["action"]
	version := txM["version"]

	vBucket := tx.Bucket(bucketVersion)
	txVal = vBucket.Get([]byte(repoId))
	var vM map[string]string
	if txVal == nil || len(txVal) == 0 {
		//new records
		vM = make(map[string]string)
	} else {
		err = json.Unmarshal(txVal, &vM)
		if err != nil {
			logger.Errorw("faild to unmarshall version content", "data", string(txVal),
				"err", err, "tx", txId)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	vM[action] = version
	txVal, err = json.Marshal(vM)
	if err != nil {
		logger.Errorw("failed to marshal version content", "data", fmt.Sprintf("%v", vM),
			"err", err, "tx", txId)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = vBucket.Put([]byte(repoId), txVal)
	if err != nil {
		logger.Errorw("faild to put version content to database", "err", err, "tx", txId)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = txBucket.Delete([]byte(txId))
	if err != nil {
		logger.Errorw("failed to delete transaction record", "err", err, "tx", txId)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		logger.Errorw("failed to commit transaction", "err", err, "tx", txId)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

var colors = map[string]string{
	"release": "#2fa84a",
	"latest":  "#2e4ea2",
	"nightly": "#8d4bae",
	"patch":   "#b45853",
}

func getVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	versionType := strings.TrimSpace(query.Get("version_type"))
	repoKey := strings.TrimSpace(query.Get("repo"))
	if versionType == "" {
		versionType = "latest"
	}

	if repoKey == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	value := "n/a"
	err := db.View(func(t *bbolt.Tx) error {
		b := t.Bucket(bucketVersion)
		data := b.Get([]byte(repoKey))
		if data == nil || len(data) == 0 {
			return nil
		}
		var m map[string]string
		err := json.Unmarshal(data, &m)
		if err != nil {
			return err
		}
		v, ok := m[versionType]
		if !ok {
			return nil
		}
		value = strings.TrimSpace(v)
		return nil
	})
	if err != nil {
		logger.Warnw("failed to load version information", "err", err, "repo", repoKey,
			"version_type", versionType)
	}
	var buf bytes.Buffer
	color, ok := colors[strings.ToLower(versionType)]
	if !ok {
		color = "#5272B4"
	}
	err = badge.Render(versionType, value, badge.Color(color), &buf)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache,max-age=0")
	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}
