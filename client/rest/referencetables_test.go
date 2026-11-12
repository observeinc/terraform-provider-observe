package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const referenceTableCSV = "id,name\n1,first\n2,second\n"

func writeReferenceTableCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "my table.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func newReferenceTableInput(sourceFilePath string) *ReferenceTableInput {
	label := "my table"
	return &ReferenceTableInput{
		Metadata:       ReferenceTableMetadataInput{Label: &label, PrimaryKey: &[]string{"id"}},
		SourceFilePath: sourceFilePath,
		Schema:         []ReferenceTableSchemaInput{{Name: "id", Type: "int64"}, {Name: "name", Type: "string"}},
	}
}

// bufferedReferenceTableBody encodes the request the way the provider did
// before uploads were streamed, using the given boundary.
func bufferedReferenceTableBody(t *testing.T, input *ReferenceTableInput, boundary string) []byte {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.SetBoundary(boundary); err != nil {
		t.Fatal(err)
	}
	uploadPart, err := writer.CreateFormFile("upload", filepath.Base(input.SourceFilePath))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(input.SourceFilePath)
	if err != nil {
		t.Fatal(err)
	}
	uploadPart.Write(content)
	schemaPart, err := writer.CreateFormFile("schema", "schema.json")
	if err != nil {
		t.Fatal(err)
	}
	schema, _ := json.Marshal(input.Schema)
	schemaPart.Write(schema)
	metadata, _ := json.Marshal(input.Metadata)
	writer.WriteField("metadata", string(metadata))
	writer.Close()
	return body.Bytes()
}

func readUploadBody(t *testing.T, upload *MultipartFileUpload) []byte {
	t.Helper()
	body, err := upload.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestReferenceTableRequestBodyMatchesBufferedEncoding(t *testing.T) {
	input := newReferenceTableInput(writeReferenceTableCSV(t, referenceTableCSV))
	upload, err := input.RequestBody()
	if err != nil {
		t.Fatal(err)
	}
	mediaType, params, err := mime.ParseMediaType(upload.ContentType())
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("content type = %q (%v), want multipart/form-data", upload.ContentType(), err)
	}

	streamed := readUploadBody(t, upload)
	if want := bufferedReferenceTableBody(t, input, params["boundary"]); !bytes.Equal(streamed, want) {
		t.Errorf("streamed body differs from the buffered encoding\n got: %q\nwant: %q", streamed, want)
	}
	if int64(len(streamed)) != upload.ContentLength() {
		t.Errorf("ContentLength() = %d, body has %d bytes", upload.ContentLength(), len(streamed))
	}
	if again := readUploadBody(t, upload); !bytes.Equal(again, streamed) {
		t.Errorf("second Open() returned a different body, so a retry would not resend the same request")
	}
}

func TestReferenceTableRequestBodyEmptyFile(t *testing.T) {
	input := newReferenceTableInput(writeReferenceTableCSV(t, ""))
	upload, err := input.RequestBody()
	if err != nil {
		t.Fatal(err)
	}
	_, params, _ := mime.ParseMediaType(upload.ContentType())
	if streamed := readUploadBody(t, upload); !bytes.Equal(streamed, bufferedReferenceTableBody(t, input, params["boundary"])) {
		t.Errorf("empty file body differs from the buffered encoding: %q", streamed)
	}
}

func TestReferenceTableRequestBodyRejectsUnusableSourceFile(t *testing.T) {
	if _, err := newReferenceTableInput(filepath.Join(t.TempDir(), "missing.csv")).RequestBody(); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: err = %v, want fs.ErrNotExist", err)
	}
	if _, err := newReferenceTableInput(t.TempDir()).RequestBody(); err == nil {
		t.Error("directory: err = nil, want an error")
	}
}

func TestReferenceTableUploadDetectsFileSizeChange(t *testing.T) {
	for _, tc := range []struct {
		name         string
		changeBefore bool
		newContent   string
	}{
		{name: "shrank before open", changeBefore: true, newContent: "id\n"},
		{name: "grew before open", changeBefore: true, newContent: referenceTableCSV + "3,third\n"},
		{name: "shrank while reading", newContent: "id\n"},
		{name: "grew while reading", newContent: referenceTableCSV + "3,third\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceFilePath := writeReferenceTableCSV(t, referenceTableCSV)
			upload, err := newReferenceTableInput(sourceFilePath).RequestBody()
			if err != nil {
				t.Fatal(err)
			}
			rewrite := func() {
				if err := os.WriteFile(sourceFilePath, []byte(tc.newContent), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.changeBefore {
				rewrite()
				if _, err := upload.Open(); !errors.Is(err, errSourceFileChangedSize) {
					t.Errorf("Open: err = %v, want errSourceFileChangedSize", err)
				}
				return
			}
			body, err := upload.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer body.Close()
			rewrite()
			if _, err := io.ReadAll(body); !errors.Is(err, errSourceFileChangedSize) {
				t.Errorf("read: err = %v, want errSourceFileChangedSize", err)
			}
		})
	}
}

// assertFileClosed waits for every descriptor on path to close; the HTTP
// transport may close a request body asynchronously after Do returns.
func assertFileClosed(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		return
	}
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	isOpen := func() bool {
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name())); err == nil && target == path {
				return true
			}
		}
		return false
	}
	for deadline := time.Now().Add(2 * time.Second); isOpen(); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%s is still open", path)
		}
	}
}

func TestReferenceTableUploadSendsSameMultipartRequest(t *testing.T) {
	sourceFilePath := writeReferenceTableCSV(t, referenceTableCSV)
	input := newReferenceTableInput(sourceFilePath)
	var gotMethods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethods = append(gotMethods, r.Method)
		if len(r.TransferEncoding) != 0 || r.ContentLength <= 0 {
			t.Errorf("%s: transfer encoding %v, Content-Length %d; want a fixed Content-Length", r.Method, r.TransferEncoding, r.ContentLength)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("%s: ParseMultipartForm: %v", r.Method, err)
		}
		upload, uploadHeader, err := r.FormFile("upload")
		if err != nil {
			t.Fatalf("upload part: %v", err)
		}
		content, _ := io.ReadAll(upload)
		if uploadHeader.Filename != "my table.csv" || uploadHeader.Header.Get("Content-Type") != "application/octet-stream" || string(content) != referenceTableCSV {
			t.Errorf("upload part: filename %q, content type %q, content %q", uploadHeader.Filename, uploadHeader.Header.Get("Content-Type"), content)
		}
		schema, schemaHeader, err := r.FormFile("schema")
		if err != nil {
			t.Fatalf("schema part: %v", err)
		}
		schemaContent, _ := io.ReadAll(schema)
		if schemaHeader.Filename != "schema.json" || string(schemaContent) != `[{"name":"id","type":"int64"},{"name":"name","type":"string"}]` {
			t.Errorf("schema part: filename %q, content %q", schemaHeader.Filename, schemaContent)
		}
		if metadata := r.FormValue("metadata"); metadata != `{"label":"my table","primaryKey":["id"]}` {
			t.Errorf("metadata = %q", metadata)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"41000003","label":"my table"}`))
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	if refTable, err := client.CreateReferenceTable(context.Background(), input); err != nil || refTable.Id != "41000003" {
		t.Fatalf("CreateReferenceTable = %+v, %v", refTable, err)
	}
	if refTable, err := client.UpdateReferenceTable(context.Background(), "41000003", input); err != nil || refTable.Id != "41000003" {
		t.Fatalf("UpdateReferenceTable = %+v, %v", refTable, err)
	}
	if len(gotMethods) != 2 || gotMethods[0] != http.MethodPost || gotMethods[1] != http.MethodPut {
		t.Errorf("methods = %v, want [POST PUT]", gotMethods)
	}
	assertFileClosed(t, sourceFilePath)
}

func TestReferenceTableUploadClosesFileOnServerError(t *testing.T) {
	sourceFilePath := writeReferenceTableCSV(t, referenceTableCSV)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad schema"}`))
	}))
	defer server.Close()

	_, err := New(server.URL, server.Client()).CreateReferenceTable(context.Background(), newReferenceTableInput(sourceFilePath))
	if !HasStatusCode(err, http.StatusBadRequest) {
		t.Errorf("err = %v, want 400", err)
	}
	assertFileClosed(t, sourceFilePath)
}

func TestReferenceTableUploadReturnsFileReadError(t *testing.T) {
	const sourceFileSize = 64 << 20
	sourceFilePath := filepath.Join(t.TempDir(), "table.csv")
	if err := os.WriteFile(sourceFilePath, bytes.Repeat([]byte("1,a\n"), sourceFileSize/4), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The file is far larger than the socket buffers, so the client is
		// still reading it when it gets truncated.
		if err := os.Truncate(sourceFilePath, 0); err != nil {
			t.Errorf("truncate: %v", err)
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := New(server.URL, server.Client()).CreateReferenceTable(context.Background(), newReferenceTableInput(sourceFilePath))
	if !errors.Is(err, errSourceFileChangedSize) {
		t.Errorf("err = %v, want errSourceFileChangedSize", err)
	}
	assertFileClosed(t, sourceFilePath)
}
