package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lobo235/filesystem-gateway/internal/nfs"
)

// chmodModeRegex permits a 3-digit octal mode with an optional leading zero (e.g. "0770", "770").
var chmodModeRegex = regexp.MustCompile(`^0?[0-7]{3}$`)

const (
	maxJSONBodySize = 64 * 1024 // 64KB for most JSON request bodies
)

// limitBody wraps r.Body with http.MaxBytesReader to enforce a body size limit.
func limitBody(w http.ResponseWriter, r *http.Request, maxBytes int64) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
}

// --- Server management handlers ---

func (s *Server) listServersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servers, err := s.nfs.ListServers()
		if err != nil {
			s.log.Error("list servers failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list servers")
			return
		}
		if servers == nil {
			servers = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"servers": servers})
	}
}

func (s *Server) createServerHandler() http.HandlerFunc {
	type request struct {
		Name string `json:"name"`
		UID  int    `json:"uid"`
		GID  int    `json:"gid"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		limitBody(w, r, maxJSONBodySize)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "name is required")
			return
		}
		if !validServerName(req.Name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "server name must match ^[a-z0-9][a-z0-9-]{0,47}(/[a-z0-9][a-z0-9-]{0,47}){0,2}$")
			return
		}
		if err := s.nfs.CreateServer(req.Name, req.UID, req.GID); err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			s.log.Error("create server failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create server")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "status": "created"})
	}
}

func (s *Server) deleteServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		if r.URL.Query().Get("confirm") != "true" {
			writeError(w, http.StatusBadRequest, "missing_fields", "confirm=true query parameter is required")
			return
		}
		if err := s.nfs.DeleteServer(name); err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			s.log.Error("delete server failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete server")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"name": name, "status": "deleted"})
	}
}

func (s *Server) statServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		info, err := s.nfs.StatServer(name)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			s.log.Error("stat server failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to stat server")
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func (s *Server) chmodServerHandler() http.HandlerFunc {
	type request struct {
		Mode string `json:"mode"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		limitBody(w, r, maxJSONBodySize)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if req.Mode == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "mode is required")
			return
		}
		if !chmodModeRegex.MatchString(req.Mode) {
			writeError(w, http.StatusBadRequest, "invalid_body", "mode must match ^0?[0-7]{3}$")
			return
		}
		// Parse as octal — regex guarantees 3 octal digits with optional leading zero.
		parsed, err := strconv.ParseUint(req.Mode, 8, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "mode must be a valid octal value")
			return
		}
		mode := os.FileMode(parsed) // #nosec G115 -- value bounded to 0o777 by regex
		if err := s.nfs.ChmodServer(name, mode); err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			s.log.Error("chmod server failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to chmod server")
			return
		}
		// Echo the canonical octal form back to the caller.
		writeJSON(w, http.StatusOK, map[string]string{
			"name":   name,
			"mode":   "0" + strings.TrimPrefix(req.Mode, "0"),
			"status": "ok",
		})
	}
}

// --- File download ---

func (s *Server) downloadHandler() http.HandlerFunc {
	type request struct {
		URL      string `json:"url"`
		DestPath string `json:"dest_path"`
		Extract  bool   `json:"extract"`
		UID      int    `json:"uid"`
		GID      int    `json:"gid"`
		Mode     string `json:"mode"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		limitBody(w, r, maxJSONBodySize)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "url is required")
			return
		}
		if !validDownloadURL(req.URL, s.downloadHosts) {
			writeError(w, http.StatusBadRequest, "invalid_body", "url domain is not allowed")
			return
		}
		if req.DestPath == "" {
			req.DestPath = "."
		}
		if req.Mode == "" {
			req.Mode = "overwrite"
		}
		if !nfs.ValidDownloadMode(req.Mode) {
			writeError(w, http.StatusBadRequest, "invalid_body", "mode must be one of: overwrite, skip_existing, clean_first")
			return
		}
		id, err := s.nfs.StartDownload(name, req.URL, req.DestPath, req.Extract, req.UID, req.GID, nfs.DownloadMode(req.Mode))
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			s.log.Error("start download failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to start download")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "running"})
	}
}

func (s *Server) getDownloadStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		downloadID := r.PathValue("downloadID")
		status, err := s.nfs.GetDownloadStatus(name, downloadID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid download ID") {
				writeError(w, http.StatusNotFound, "not_found", "download not found")
				return
			}
			s.log.Error("get download status failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get download status")
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

// --- Disk usage ---

func (s *Server) diskUsageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		size, err := s.nfs.DiskUsage(name)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			s.log.Error("disk usage failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get disk usage")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"server": name, "bytes": size})
	}
}

// --- File operations ---

func (s *Server) listFilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		subPath := r.URL.Query().Get("path")
		files, err := s.nfs.ListFiles(name, subPath)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			s.log.Error("list files failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list files")
			return
		}
		if files == nil {
			files = []nfs.FileEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"files": files})
	}
}

func (s *Server) readFileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		subPath := r.URL.Query().Get("path")
		if subPath == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "path query parameter is required")
			return
		}
		data, err := s.nfs.ReadFile(name, subPath)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "exceeds maximum") {
				writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
				return
			}
			if strings.Contains(err.Error(), "no such file") {
				writeError(w, http.StatusNotFound, "not_found", "file not found")
				return
			}
			s.log.Error("read file failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to read file")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"content": string(data)})
	}
}

func (s *Server) grepFilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		q := r.URL.Query()
		subPath := q.Get("path")
		pattern := q.Get("pattern")
		if pattern == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "pattern query parameter is required")
			return
		}

		opts := nfs.GrepOpts{}
		if v := q.Get("case_insensitive"); v != "" {
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid_body", "case_insensitive must be a boolean")
				return
			}
			opts.CaseInsensitive = parsed
		}
		if v := q.Get("skip_exts"); v != "" {
			opts.SkipExts = splitCSV(v)
		}

		result, err := s.nfs.GrepFiles(name, subPath, pattern, opts)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			s.log.Error("grep files failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to grep files")
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// findFilesHandler handles GET /servers/{name}/files/find.
//
// Query params (all optional except `name` from path):
//   - path: subdirectory to walk (defaults to server root)
//   - name_glob: filepath.Match pattern against each entry's basename
//   - type: "f" (files only), "d" (dirs only), empty (both)
//   - max_depth: integer, negative = unlimited (default -1)
//   - modified_since: RFC3339 timestamp; drops earlier entries
//   - skip_exts: comma-separated extensions to drop from results
//   - max_entries: positive integer, capped at lsHardMaxEntries
func (s *Server) findFilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		q := r.URL.Query()

		opts := nfs.FindOpts{}
		opts.NameGlob = q.Get("name_glob")
		opts.Type = q.Get("type")
		switch opts.Type {
		case "", "f", "d":
		default:
			writeError(w, http.StatusBadRequest, "invalid_body", `type must be "f", "d", or empty`)
			return
		}
		if v := q.Get("max_depth"); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid_body", "max_depth must be an integer")
				return
			}
			opts.MaxDepth = parsed
		}
		if v := q.Get("modified_since"); v != "" {
			parsed, err := time.Parse(time.RFC3339, v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid_body", "modified_since must be RFC3339")
				return
			}
			opts.ModifiedSince = parsed
		}
		if v := q.Get("skip_exts"); v != "" {
			opts.SkipExts = splitCSV(v)
		}
		opts.MaxEntries = lsDefaultMaxEntries
		if v := q.Get("max_entries"); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil || parsed <= 0 {
				writeError(w, http.StatusBadRequest, "invalid_body", "max_entries must be a positive integer")
				return
			}
			opts.MaxEntries = parsed
		}
		if opts.MaxEntries > lsHardMaxEntries {
			opts.MaxEntries = lsHardMaxEntries
		}

		subPath := q.Get("path")
		result, err := s.nfs.FindFiles(name, subPath, opts)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "invalid name_glob") || strings.Contains(err.Error(), "invalid type") {
				writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "server_not_found", err.Error())
				return
			}
			if strings.Contains(err.Error(), "path not found") {
				writeError(w, http.StatusNotFound, "path_not_found", err.Error())
				return
			}
			s.log.Error("find files failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to find files")
			return
		}
		if result.Entries == nil {
			result.Entries = []nfs.DirEntry{}
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// splitCSV splits a comma-separated value, trimming whitespace and dropping empties.
func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// --- Backup operations ---

func (s *Server) listBackupsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		backups, err := s.nfs.ListBackups(name)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			s.log.Error("list backups failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list backups")
			return
		}
		if backups == nil {
			backups = []nfs.BackupInfo{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"backups": backups})
	}
}

func (s *Server) startBackupHandler() http.HandlerFunc {
	type request struct {
		UID int `json:"uid"`
		GID int `json:"gid"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		limitBody(w, r, maxJSONBodySize)
		var req request
		// Best-effort decode — empty body is fine, uid/gid default to 0.
		_ = json.NewDecoder(r.Body).Decode(&req)
		id, err := s.nfs.StartBackup(name, req.UID, req.GID)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			s.log.Error("start backup failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to start backup")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"server": name, "backup_id": id, "status": "running"})
	}
}

func (s *Server) getBackupStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		backupID := r.PathValue("backupID")
		status, err := s.nfs.GetBackupStatus(name, backupID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeError(w, http.StatusNotFound, "not_found", "backup not found")
				return
			}
			s.log.Error("get backup status failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get backup status")
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

// --- Restore and migrate ---

func (s *Server) restoreHandler() http.HandlerFunc {
	type request struct {
		BackupID string `json:"backup_id"`
		UID      int    `json:"uid"`
		GID      int    `json:"gid"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		limitBody(w, r, maxJSONBodySize)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if req.BackupID == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "backup_id is required")
			return
		}
		if err := s.nfs.Restore(name, req.BackupID, req.UID, req.GID); err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "backup not found") {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			s.log.Error("restore failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to restore backup")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"server": name, "backup_id": req.BackupID, "status": "restored"})
	}
}

func (s *Server) migrateHandler() http.HandlerFunc {
	type request struct {
		NewName string `json:"new_name"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		limitBody(w, r, maxJSONBodySize)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if req.NewName == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "new_name is required")
			return
		}
		if !validServerName(req.NewName) {
			writeError(w, http.StatusBadRequest, "invalid_body", "new_name must match ^[a-z0-9][a-z0-9-]{0,47}(/[a-z0-9][a-z0-9-]{0,47}){0,2}$")
			return
		}
		if err := s.nfs.Migrate(name, req.NewName); err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			if strings.Contains(err.Error(), "already exists") {
				writeError(w, http.StatusConflict, "conflict", err.Error())
				return
			}
			s.log.Error("migrate failed", "error", err, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to migrate server")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"old_name": name, "new_name": req.NewName, "status": "migrated"})
	}
}

// --- Archive contents ---

func (s *Server) archiveContentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		archivePath := r.URL.Query().Get("path")
		if archivePath == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "path query parameter is required")
			return
		}
		entries, err := s.nfs.ListArchiveContents(name, archivePath)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not found") {
				writeError(w, http.StatusNotFound, "not_found", "archive not found")
				return
			}
			s.log.Error("list archive contents failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list archive contents")
			return
		}
		if entries == nil {
			entries = []nfs.ArchiveEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
	}
}

// --- Write file ---

func (s *Server) writeFileHandler() http.HandlerFunc {
	type request struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		UID     int    `json:"uid"`
		GID     int    `json:"gid"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}
		limitBody(w, r, s.nfs.MaxWriteFileSize()+4096)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if req.Path == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "path is required")
			return
		}
		if err := s.nfs.WriteFile(name, req.Path, req.Content, req.UID, req.GID); err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "exceeds maximum") {
				writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
				return
			}
			s.log.Error("write file failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to write file")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "path": req.Path})
	}
}

// moveFileHandler handles POST /servers/{name}/files/move.
func (s *Server) moveFileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_name", "invalid server name")
			return
		}

		limitBody(w, r, maxJSONBodySize)
		var body struct {
			SrcPath string `json:"src_path"`
			DstPath string `json:"dst_path"`
			UID     int    `json:"uid"`
			GID     int    `json:"gid"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
			return
		}
		if body.SrcPath == "" || body.DstPath == "" {
			writeError(w, http.StatusBadRequest, "missing_param", "src_path and dst_path are required")
			return
		}

		if err := s.nfs.MoveFile(name, body.SrcPath, body.DstPath, body.UID, body.GID); err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeError(w, http.StatusNotFound, "not_found", err.Error())
				return
			}
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path escapes server directory")
				return
			}
			s.log.Error("move file failed", "server", name, "error", err)
			writeError(w, http.StatusInternalServerError, "internal", "failed to move file")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// deleteFileHandler handles DELETE /servers/{name}/files/delete?path=...
func (s *Server) deleteFileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_name", "invalid server name")
			return
		}

		path := r.URL.Query().Get("path")
		if path == "" {
			writeError(w, http.StatusBadRequest, "missing_param", "path query parameter is required")
			return
		}

		fullPath, err := s.nfs.SafePath(name, path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "path_traversal", "path escapes server directory")
			return
		}

		// Prevent deletion of the server root directory.
		serverRoot, _ := s.nfs.SafePath(name)
		if fullPath == serverRoot {
			writeError(w, http.StatusBadRequest, "invalid_path", "cannot delete server root directory")
			return
		}

		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			writeError(w, http.StatusNotFound, "not_found", "file or directory not found")
			return
		}

		if err := os.RemoveAll(fullPath); err != nil {
			s.log.Error("delete file failed", "server", name, "path", path, "error", err)
			writeError(w, http.StatusInternalServerError, "internal", "failed to delete")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// --- Read-only listing and file reads ---

const (
	lsDefaultMaxEntries = 1000
	lsHardMaxEntries    = 10000
	readDefaultMaxBytes = 1 << 20        // 1 MiB
	readHardMaxBytes    = 10 * (1 << 20) // 10 MiB
)

// lsHandler handles GET /servers/{name}/ls.
func (s *Server) lsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}

		q := r.URL.Query()
		subPath := q.Get("path")

		recursive := false
		if v := q.Get("recursive"); v != "" {
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid_body", "recursive must be a boolean")
				return
			}
			recursive = parsed
		}

		maxEntries := lsDefaultMaxEntries
		if v := q.Get("max_entries"); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil || parsed <= 0 {
				writeError(w, http.StatusBadRequest, "invalid_body", "max_entries must be a positive integer")
				return
			}
			maxEntries = parsed
		}
		if maxEntries > lsHardMaxEntries {
			maxEntries = lsHardMaxEntries
		}

		entries, truncated, err := s.nfs.ListDir(name, subPath, recursive, maxEntries)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "server_not_found", err.Error())
				return
			}
			if strings.Contains(err.Error(), "path not found") {
				writeError(w, http.StatusNotFound, "path_not_found", err.Error())
				return
			}
			s.log.Error("list dir failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list directory")
			return
		}
		if entries == nil {
			entries = []nfs.DirEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "truncated": truncated})
	}
}

// readHandler handles GET /servers/{name}/read.
func (s *Server) readHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validServerName(name) {
			writeError(w, http.StatusBadRequest, "invalid_body", "invalid server name")
			return
		}

		q := r.URL.Query()
		subPath := q.Get("path")
		if subPath == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "path query parameter is required")
			return
		}

		maxBytes := int64(readDefaultMaxBytes)
		if v := q.Get("max_bytes"); v != "" {
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil || parsed <= 0 {
				writeError(w, http.StatusBadRequest, "invalid_body", "max_bytes must be a positive integer")
				return
			}
			if parsed > readHardMaxBytes {
				writeError(w, http.StatusRequestEntityTooLarge, "max_bytes_exceeded", "max_bytes exceeds hard cap of 10485760 bytes (10 MiB)")
				return
			}
			maxBytes = parsed
		}

		origin := q.Get("origin")
		if origin == "" {
			origin = "start"
		}
		var originEnd bool
		switch origin {
		case "start":
			originEnd = false
		case "end":
			originEnd = true
		default:
			writeError(w, http.StatusBadRequest, "invalid_body", "origin must be 'start' or 'end'")
			return
		}

		data, truncated, err := s.nfs.ReadFileEx(name, subPath, maxBytes, originEnd)
		if err != nil {
			if errors.Is(err, nfs.ErrPathTraversal) {
				writeError(w, http.StatusBadRequest, "path_traversal", "path traversal detected")
				return
			}
			if errors.Is(err, nfs.ErrNotRegularFile) {
				writeError(w, http.StatusUnsupportedMediaType, "not_regular_file", "target is not a regular file (device, socket, or named pipe)")
				return
			}
			if errors.Is(err, nfs.ErrFileNotFound) {
				writeError(w, http.StatusNotFound, "file_not_found", "file not found")
				return
			}
			if strings.Contains(err.Error(), "server not found") {
				writeError(w, http.StatusNotFound, "server_not_found", err.Error())
				return
			}
			s.log.Error("read file failed", "error", err, "server", name, "trace_id", traceIDFromContext(r.Context()))
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to read file")
			return
		}

		contentType := "application/octet-stream"
		if utf8.Valid(data) {
			contentType = "text/plain; charset=utf-8"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		if truncated {
			w.Header().Set("X-Truncated", "true")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
