package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/locngoxuan/xsql"
	"github.com/narqo/go-badge"
)

type (
	VersionRequest struct {
		Namespace     string `json:"namespace,omitempty"`
		RepoId        string `json:"repo_id,omitempty"`
		ReleaseAction string `json:"action,omitempty"`
		Version       string `json:"version,omitempty"`
	}
)

func rollbackVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request VersionRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.Errorw("failed to decode request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vtype := versionDevelopment
	if v := strings.TrimSpace(request.ReleaseAction); v != "" {
		vtype = v
	}

	if _, ok := vTyps[vtype]; !ok {
		logger.Errorw("action is not supported", "action", vtype)
		http.Error(w, "wrong version type", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.RepoId) == "" ||
		strings.TrimSpace(request.Namespace) == "" ||
		strings.TrimSpace(request.Version) == "" {
		logger.Error("repo_id or version is missing")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	err = deleteVersion(strings.TrimSpace(request.Namespace), strings.TrimSpace(request.RepoId), vtype)
	if err != nil {
		logger.Errorw("failed to put data to database", "err", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func updateVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request VersionRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.Errorw("failed to decode request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vtype := versionDevelopment
	if v := strings.TrimSpace(request.ReleaseAction); v != "" {
		vtype = v
	}

	if _, ok := vTyps[vtype]; !ok {
		logger.Errorw("action is not supported", "action", vtype)
		http.Error(w, "wrong version type", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.RepoId) == "" ||
		strings.TrimSpace(request.Namespace) == "" ||
		strings.TrimSpace(request.Version) == "" {
		logger.Error("repo_id or version is missing")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	txId := NewObjectID().Hex()

	entity := VersionEntity{
		Id:        txId,
		Namespace: strings.TrimSpace(request.Namespace),
		RepoId:    strings.TrimSpace(request.RepoId),
		Type:      vtype,
		Value:     strings.TrimSpace(request.Version),
		Status:    statusCommitted,
		Created:   time.Now(),
	}
	err = xsql.Insert(entity)
	if err != nil {
		logger.Errorw("failed to put data to database", "err", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(txId))
}

var colors = map[string]string{
	versionRelease:     "#2fa84a",
	versionDevelopment: "#2e4ea2",
	versionNightly:     "#8d4bae",
	versionPatch:       "#b45853",
}

func getAllRepos(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	items, err := findAllRepos()
	if err != nil {
		logger.Errorw("failed to find all repo", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(items)
	if err != nil {
		logger.Errorw("failed to marshal response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-cache,max-age=0")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func getVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	versionType := strings.TrimSpace(query.Get("type"))
	namespace := strings.TrimSpace(query.Get("namespace"))
	repoKey := strings.ToLower(strings.TrimSpace(query.Get("repo")))
	if versionType == "" {
		versionType = versionDevelopment
	}

	if repoKey == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	value := "n/a"
	ver, err := findVersion(namespace, repoKey, versionType)
	if err != nil {
		if err != nil {
			logger.Warnw("failed to load version information", "err", err, "repo", repoKey,
				"version_type", versionType)
		}
		if err != xsql.ErrNotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		value = ver.Value
	}
	var buf bytes.Buffer
	color, ok := colors[versionType]
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
