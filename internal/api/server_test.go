package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lobo235/filesystem-gateway/internal/nfs"
)

// --- Mock NFS client ---

type mockNFS struct {
	servers          []string
	listErr          error
	createErr        error
	deleteErr        error
	statInfo         *nfs.ServerInfo
	statErr          error
	chmodErr         error
	diskUsage        int64
	diskUsageErr     error
	files            []nfs.FileEntry
	listFilesErr     error
	fileContent      []byte
	readFileErr      error
	grepResult       *nfs.GrepResult
	grepErr          error
	backups          []nfs.BackupInfo
	listBackErr      error
	backupID         string
	startBackErr     error
	backupStatus     *nfs.BackupStatus
	getStatusErr     error
	restoreErr       error
	migrateErr       error
	downloadID       string
	startDownloadErr error
	downloadStatus   *nfs.DownloadStatus
	getDownloadErr   error
	archiveEntries   []nfs.ArchiveEntry
	archiveErr       error
	writeFileErr     error
	moveFileErr      error
}

func (m *mockNFS) SafePath(parts ...string) (string, error) { return "", nil }
func (m *mockNFS) ListServers() ([]string, error)           { return m.servers, m.listErr }
func (m *mockNFS) CreateServer(string, int, int) error      { return m.createErr }
func (m *mockNFS) DeleteServer(string) error                { return m.deleteErr }
func (m *mockNFS) StatServer(string) (*nfs.ServerInfo, error) {
	return m.statInfo, m.statErr
}
func (m *mockNFS) ChmodServer(string, os.FileMode) error { return m.chmodErr }
func (m *mockNFS) DiskUsage(string) (int64, error)       { return m.diskUsage, m.diskUsageErr }
func (m *mockNFS) ListFiles(string, string) ([]nfs.FileEntry, error) {
	return m.files, m.listFilesErr
}
func (m *mockNFS) ReadFile(string, string) ([]byte, error) { return m.fileContent, m.readFileErr }
func (m *mockNFS) GrepFiles(string, string, string) (*nfs.GrepResult, error) {
	return m.grepResult, m.grepErr
}
func (m *mockNFS) ListBackups(string) ([]nfs.BackupInfo, error) { return m.backups, m.listBackErr }
func (m *mockNFS) StartBackup(string, int, int) (string, error) { return m.backupID, m.startBackErr }
func (m *mockNFS) GetBackupStatus(string, string) (*nfs.BackupStatus, error) {
	return m.backupStatus, m.getStatusErr
}
func (m *mockNFS) Restore(string, string, int, int) error { return m.restoreErr }
func (m *mockNFS) Migrate(string, string) error           { return m.migrateErr }
func (m *mockNFS) StartDownload(string, string, string, bool, int, int, nfs.DownloadMode) (string, error) {
	return m.downloadID, m.startDownloadErr
}
func (m *mockNFS) GetDownloadStatus(string, string) (*nfs.DownloadStatus, error) {
	return m.downloadStatus, m.getDownloadErr
}
func (m *mockNFS) ListArchiveContents(string, string) ([]nfs.ArchiveEntry, error) {
	return m.archiveEntries, m.archiveErr
}
func (m *mockNFS) WriteFile(string, string, string, int, int) error {
	return m.writeFileErr
}
func (m *mockNFS) MoveFile(string, string, string, int, int) error {
	return m.moveFileErr
}
func (m *mockNFS) MaxWriteFileSize() int64 { return 1048576 }

// --- Helpers ---

func newTestServer(nfsClient nfs.Client) *Server {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewServer(nfsClient, "test-api-key", "test", log, DefaultDownloadHosts)
}

func doRequest(handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func doRequestNoAuth(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// --- Health ---

func TestHealthEndpoint(t *testing.T) {
	s := newTestServer(&mockNFS{})
	rr := doRequestNoAuth(s.Handler(), "GET", "/health")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var resp healthResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if resp.Version != "test" {
		t.Errorf("version = %q, want test", resp.Version)
	}
}

// --- Auth ---

func TestUnauthorized(t *testing.T) {
	s := newTestServer(&mockNFS{servers: []string{"a"}})
	rr := doRequestNoAuth(s.Handler(), "GET", "/servers")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

// --- List Servers ---

func TestListServers(t *testing.T) {
	s := newTestServer(&mockNFS{servers: []string{"mc-1", "mc-2"}})
	rr := doRequest(s.Handler(), "GET", "/servers", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var resp map[string][]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp["servers"]) != 2 {
		t.Errorf("expected 2 servers, got %d", len(resp["servers"]))
	}
}

// --- Create Server ---

func TestCreateServer(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]any{"name": "mc-new", "uid": 1000, "gid": 1000}
	rr := doRequest(s.Handler(), "POST", "/servers", body)

	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rr.Code)
	}
}

func TestCreateServerInvalidName(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]any{"name": "INVALID_NAME!", "uid": 1000, "gid": 1000}
	rr := doRequest(s.Handler(), "POST", "/servers", body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestCreateServerMissingName(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]any{"uid": 1000, "gid": 1000}
	rr := doRequest(s.Handler(), "POST", "/servers", body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

// --- Delete Server ---

func TestDeleteServer(t *testing.T) {
	s := newTestServer(&mockNFS{})
	rr := doRequest(s.Handler(), "DELETE", "/servers/mc-test?confirm=true", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestDeleteServerMissingConfirm(t *testing.T) {
	s := newTestServer(&mockNFS{})
	rr := doRequest(s.Handler(), "DELETE", "/servers/mc-test", nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

// --- Stat Server ---

func TestStatServer(t *testing.T) {
	info := &nfs.ServerInfo{
		Name:    "mc-test",
		Bytes:   2048,
		UID:     1000,
		GID:     1000,
		Mode:    "0755",
		ModTime: "2026-04-16T00:00:00Z",
	}
	s := newTestServer(&mockNFS{statInfo: info})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp nfs.ServerInfo
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Name != "mc-test" {
		t.Errorf("name = %q, want mc-test", resp.Name)
	}
	if resp.Bytes != 2048 {
		t.Errorf("bytes = %d, want 2048", resp.Bytes)
	}
	if resp.Mode != "0755" {
		t.Errorf("mode = %q, want 0755", resp.Mode)
	}
}

func TestStatServer_NotFound(t *testing.T) {
	s := newTestServer(&mockNFS{statErr: fmt.Errorf("server not found: mc-test")})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test", nil)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestStatServer_InvalidName(t *testing.T) {
	s := newTestServer(&mockNFS{})
	rr := doRequest(s.Handler(), "GET", "/servers/BAD_NAME", nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestStatServer_PathTraversal(t *testing.T) {
	s := newTestServer(&mockNFS{statErr: nfs.ErrPathTraversal})
	// Use a syntactically-valid name so we hit the handler's traversal branch
	// rather than the upstream validServerName 400.
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test", nil)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != "path_traversal" {
		t.Errorf("code = %q, want path_traversal", resp.Code)
	}
}

// --- Chmod Server ---

func TestChmodServer(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]string{"mode": "0770"}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/chmod", body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["mode"] != "0770" {
		t.Errorf("mode = %q, want 0770", resp["mode"])
	}
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want ok", resp["status"])
	}
	if resp["name"] != "mc-test" {
		t.Errorf("name = %q, want mc-test", resp["name"])
	}
}

func TestChmodServer_ModeWithoutLeadingZero(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]string{"mode": "770"}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/chmod", body)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestChmodServer_InvalidMode(t *testing.T) {
	cases := []string{"0778", "abc", "", "01000", "0-755", "9999"}
	for _, mode := range cases {
		t.Run(mode, func(t *testing.T) {
			s := newTestServer(&mockNFS{})
			body := map[string]string{"mode": mode}
			rr := doRequest(s.Handler(), "POST", "/servers/mc-test/chmod", body)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("mode=%q status = %d, want 400", mode, rr.Code)
			}
		})
	}
}

func TestChmodServer_MissingMode(t *testing.T) {
	s := newTestServer(&mockNFS{})
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/chmod", map[string]string{})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestChmodServer_NotFound(t *testing.T) {
	s := newTestServer(&mockNFS{chmodErr: fmt.Errorf("server not found: mc-test")})
	body := map[string]string{"mode": "0755"}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/chmod", body)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestChmodServer_InvalidName(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]string{"mode": "0755"}
	rr := doRequest(s.Handler(), "POST", "/servers/BAD_NAME/chmod", body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestChmodServer_PathTraversal(t *testing.T) {
	s := newTestServer(&mockNFS{chmodErr: nfs.ErrPathTraversal})
	body := map[string]string{"mode": "0755"}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/chmod", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != "path_traversal" {
		t.Errorf("code = %q, want path_traversal", resp.Code)
	}
}

// --- Disk Usage ---

func TestDiskUsage(t *testing.T) {
	s := newTestServer(&mockNFS{diskUsage: 1024000})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/disk-usage", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

// --- List Files ---

func TestListFiles(t *testing.T) {
	files := []nfs.FileEntry{
		{Name: "world", IsDir: true, Size: 4096},
	}
	s := newTestServer(&mockNFS{files: files})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/files?path=.", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

// --- Read File ---

func TestReadFile(t *testing.T) {
	s := newTestServer(&mockNFS{fileContent: []byte("log data")})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/files/read?path=logs/latest.log", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestReadFileMissingPath(t *testing.T) {
	s := newTestServer(&mockNFS{})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/files/read", nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

// --- Grep ---

func TestGrepFiles(t *testing.T) {
	result := &nfs.GrepResult{Lines: []string{"match1"}, Count: 1}
	s := newTestServer(&mockNFS{grepResult: result})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/files/grep?path=logs&pattern=ERROR", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestGrepFilesMissingPattern(t *testing.T) {
	s := newTestServer(&mockNFS{})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/files/grep?path=logs", nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

// --- Backups ---

func TestListBackups(t *testing.T) {
	backups := []nfs.BackupInfo{{ID: "2026-03-22T10-00-00", Size: 1024}}
	s := newTestServer(&mockNFS{backups: backups})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/backups", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestStartBackup(t *testing.T) {
	s := newTestServer(&mockNFS{backupID: "2026-03-22T10-00-00"})
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/backups", nil)

	if rr.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", rr.Code)
	}
}

func TestGetBackupStatus(t *testing.T) {
	status := &nfs.BackupStatus{Server: "mc-test", ID: "abc", Status: "done"}
	s := newTestServer(&mockNFS{backupStatus: status})
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/backups/abc", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

// --- Restore ---

func TestRestore(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]string{"backup_id": "2026-03-22T10-00-00"}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/restore", body)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestRestoreMissingBackupID(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]string{}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/restore", body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

// --- Migrate ---

func TestMigrate(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]string{"new_name": "mc-new"}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/migrate", body)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestMigrateInvalidNewName(t *testing.T) {
	s := newTestServer(&mockNFS{})
	body := map[string]string{"new_name": "INVALID!"}
	rr := doRequest(s.Handler(), "POST", "/servers/mc-test/migrate", body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

// --- Server name validation across endpoints ---

func TestInvalidServerNameRejected(t *testing.T) {
	s := newTestServer(&mockNFS{})
	handler := s.Handler()

	endpoints := []struct {
		method string
		path   string
	}{
		{"DELETE", "/servers/BAD_NAME?confirm=true"},
		{"GET", "/servers/BAD_NAME/disk-usage"},
		{"GET", "/servers/BAD_NAME/files"},
		{"GET", "/servers/BAD_NAME/files/read?path=x"},
		{"GET", "/servers/BAD_NAME/files/grep?path=x&pattern=y"},
		{"GET", "/servers/BAD_NAME/backups"},
		{"POST", "/servers/BAD_NAME/backups"},
		{"GET", "/servers/BAD_NAME/backups/id"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			rr := doRequest(handler, ep.method, ep.path, nil)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 for %s %s", rr.Code, ep.method, ep.path)
			}
		})
	}
}

// --- Path traversal through handler ---

func TestPathTraversalInHandler(t *testing.T) {
	mn := &mockNFS{
		readFileErr: nfs.ErrPathTraversal,
	}
	s := newTestServer(mn)
	rr := doRequest(s.Handler(), "GET", "/servers/mc-test/files/read?path=../../etc/passwd", nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
	var resp errorResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != "path_traversal" {
		t.Errorf("code = %q, want path_traversal", resp.Code)
	}
}

// --- ListServers error path ---

func TestListServers_Error(t *testing.T) {
	s := newTestServer(&mockNFS{listErr: fmt.Errorf("disk error")})
	rr := doRequest(s.Handler(), "GET", "/servers", nil)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}

func TestListServers_NilReturnsEmptyArray(t *testing.T) {
	s := newTestServer(&mockNFS{servers: nil})
	rr := doRequest(s.Handler(), "GET", "/servers", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var resp map[string][]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["servers"] == nil {
		t.Error("expected non-nil servers array")
	}
	if len(resp["servers"]) != 0 {
		t.Errorf("expected 0 servers, got %d", len(resp["servers"]))
	}
}

// --- CreateServer error paths ---

func TestCreateServer_PathTraversal(t *testing.T) {
	s := newTestServer(&mockNFS{createErr: nfs.ErrPathTraversal})
	body := map[string]any{"name": "mc-test", "uid": 1000, "gid": 1000}
	rr := doRequest(s.Handler(), "POST", "/servers", body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestCreateServer_InternalError(t *testing.T) {
	s := newTestServer(&mockNFS{createErr: fmt.Errorf("disk full")})
	body := map[string]any{"name": "mc-test", "uid": 1000, "gid": 1000}
	rr := doRequest(s.Handler(), "POST", "/servers", body)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}
