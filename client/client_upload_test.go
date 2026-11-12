package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/rest"
)

func newMiddlewareRestClient(t *testing.T, serverURL string) *rest.Client {
	t.Helper()
	observeClient := &Client{Config: &Config{RetryCount: 2, RetryWait: time.Millisecond}}
	httpClient := &http.Client{Transport: observeClient.withMiddleware(http.DefaultTransport.(*http.Transport).Clone())}
	t.Cleanup(httpClient.CloseIdleConnections)
	return rest.New(serverURL, httpClient)
}

func discardLogOutput(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

type receivedUpload struct {
	contentLength    int64
	transferEncoding []string
	bodyBytesRead    int64
	uploadFileName   string
	uploadSHA256     [32]byte
	uploadBytes      int64
	schema           string
	metadata         string
}

type countingReader struct {
	reader    io.Reader
	bytesRead int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += int64(n)
	return n, err
}

// readMultipartUploadStreaming parses the reference table request part by part
// without buffering the file, so the server does not distort heap measurements.
func readMultipartUploadStreaming(r *http.Request) (receivedUpload, error) {
	received := receivedUpload{contentLength: r.ContentLength, transferEncoding: r.TransferEncoding}
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return received, err
	}
	body := &countingReader{reader: r.Body}
	partReader := multipart.NewReader(body, params["boundary"])
	for {
		part, err := partReader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return received, err
		}
		switch part.FormName() {
		case "upload":
			received.uploadFileName = part.FileName()
			hasher := sha256.New()
			received.uploadBytes, err = io.Copy(hasher, part)
			copy(received.uploadSHA256[:], hasher.Sum(nil))
		case "schema":
			var schema []byte
			schema, err = io.ReadAll(part)
			received.schema = string(schema)
		case "metadata":
			var metadata []byte
			metadata, err = io.ReadAll(part)
			received.metadata = string(metadata)
		}
		if err != nil {
			return received, err
		}
	}
	if _, err := io.Copy(io.Discard, body); err != nil {
		return received, err
	}
	received.bodyBytesRead = body.bytesRead
	return received, nil
}

func writeGeneratedCSV(t *testing.T, path string, sizeBytes int64) [32]byte {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var row bytes.Buffer
	for row.Len() < 1<<20 {
		row.WriteString("12345,some reference value,another column\n")
	}
	chunk := row.Bytes()[:1<<20]
	hasher := sha256.New()
	for written := int64(0); written < sizeBytes; {
		toWrite := chunk
		if remaining := sizeBytes - written; remaining < int64(len(toWrite)) {
			toWrite = toWrite[:remaining]
		}
		if _, err := file.Write(toWrite); err != nil {
			t.Fatal(err)
		}
		hasher.Write(toWrite)
		written += int64(len(toWrite))
	}
	var sum [32]byte
	copy(sum[:], hasher.Sum(nil))
	return sum
}

// heapPeakSampler records the highest HeapInuse seen while an operation runs.
type heapPeakSampler struct {
	stop     chan struct{}
	done     chan struct{}
	baseline uint64
	peak     atomic.Uint64
}

func startHeapPeakSampler() *heapPeakSampler {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	sampler := &heapPeakSampler{stop: make(chan struct{}), done: make(chan struct{}), baseline: stats.HeapInuse}
	sampler.peak.Store(stats.HeapInuse)
	go func() {
		defer close(sampler.done)
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-sampler.stop:
				return
			case <-ticker.C:
				runtime.ReadMemStats(&stats)
				if stats.HeapInuse > sampler.peak.Load() {
					sampler.peak.Store(stats.HeapInuse)
				}
			}
		}
	}()
	return sampler
}

func (s *heapPeakSampler) stopAndReportGrowth() uint64 {
	close(s.stop)
	<-s.done
	if peak := s.peak.Load(); peak > s.baseline {
		return peak - s.baseline
	}
	return 0
}

func TestReferenceTableUploadHeapDoesNotScaleWithFileSize(t *testing.T) {
	if testing.Short() {
		t.Skip("generates and uploads a large file")
	}
	discardLogOutput(t)

	const (
		sourceFileSize = 160 << 20
		maxHeapGrowth  = 32 << 20
	)
	sourceFilePath := filepath.Join(t.TempDir(), "large_table.csv")
	wantSHA256 := writeGeneratedCSV(t, sourceFilePath, sourceFileSize)

	var (
		receivedMu sync.Mutex
		received   receivedUpload
		receiveErr error
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upload, err := readMultipartUploadStreaming(r)
		receivedMu.Lock()
		received, receiveErr = upload, err
		receivedMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"41000001"}`))
	}))
	defer server.Close()
	restClient := newMiddlewareRestClient(t, server.URL)

	input := &rest.ReferenceTableInput{
		Metadata:       rest.ReferenceTableMetadataInput{Label: stringPointer("large table")},
		SourceFilePath: sourceFilePath,
		Schema:         []rest.ReferenceTableSchemaInput{{Name: "id", Type: "int64"}},
	}

	sampler := startHeapPeakSampler()
	refTable, err := restClient.CreateReferenceTable(context.Background(), input)
	heapGrowth := sampler.stopAndReportGrowth()
	t.Logf("uploaded %d MiB, peak HeapInuse growth %.1f MiB", sourceFileSize>>20, float64(heapGrowth)/(1<<20))

	if err != nil {
		t.Fatalf("CreateReferenceTable: %v", err)
	}
	if refTable.Id != "41000001" {
		t.Errorf("id = %q, want 41000001", refTable.Id)
	}
	receivedMu.Lock()
	defer receivedMu.Unlock()
	if receiveErr != nil {
		t.Fatalf("server failed to read upload: %v", receiveErr)
	}
	if len(received.transferEncoding) != 0 {
		t.Errorf("transfer encoding = %v, want none (fixed Content-Length)", received.transferEncoding)
	}
	if received.contentLength != received.bodyBytesRead {
		t.Errorf("Content-Length = %d, body had %d bytes", received.contentLength, received.bodyBytesRead)
	}
	if received.uploadBytes != sourceFileSize || received.uploadSHA256 != wantSHA256 {
		t.Errorf("upload part has %d bytes and a different checksum than the %d byte source file", received.uploadBytes, int64(sourceFileSize))
	}
	if received.uploadFileName != "large_table.csv" {
		t.Errorf("upload filename = %q, want large_table.csv", received.uploadFileName)
	}
	if received.schema != `[{"name":"id","type":"int64"}]` {
		t.Errorf("schema = %q", received.schema)
	}
	if received.metadata != `{"label":"large table"}` {
		t.Errorf("metadata = %q", received.metadata)
	}
	if heapGrowth > maxHeapGrowth {
		t.Errorf("peak heap growth %.1f MiB exceeds %d MiB for a %d MiB upload; the file is being buffered in memory",
			float64(heapGrowth)/(1<<20), maxHeapGrowth>>20, sourceFileSize>>20)
	}
}

func TestReferenceTableUploadRetryReopensSourceFile(t *testing.T) {
	discardLogOutput(t)
	sourceFilePath := filepath.Join(t.TempDir(), "table.csv")
	if err := os.WriteFile(sourceFilePath, []byte("id,name\n1,first\n2,second\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var (
		requestsMu sync.Mutex
		requests   []receivedUpload
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upload, err := readMultipartUploadStreaming(r)
		if err != nil {
			t.Errorf("server failed to read upload: %v", err)
		}
		requestsMu.Lock()
		requests = append(requests, upload)
		attempt := len(requests)
		requestsMu.Unlock()
		if attempt == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"41000002"}`))
	}))
	defer server.Close()
	restClient := newMiddlewareRestClient(t, server.URL)

	input := &rest.ReferenceTableInput{SourceFilePath: sourceFilePath}
	if _, err := restClient.UpdateReferenceTable(context.Background(), "41000002", input); err != nil {
		t.Fatalf("UpdateReferenceTable: %v", err)
	}

	requestsMu.Lock()
	defer requestsMu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("server saw %d requests, want 2 (one 429 then a retry)", len(requests))
	}
	wantSHA256 := sha256.Sum256([]byte("id,name\n1,first\n2,second\n"))
	for attempt, upload := range requests {
		if upload.uploadSHA256 != wantSHA256 || upload.contentLength != upload.bodyBytesRead {
			t.Errorf("attempt %d: upload bytes = %d, Content-Length = %d, body bytes = %d; want the full file",
				attempt+1, upload.uploadBytes, upload.contentLength, upload.bodyBytesRead)
		}
	}
}

func TestRetryResendsReplayableRequestBody(t *testing.T) {
	discardLogOutput(t)
	var (
		bodiesMu sync.Mutex
		bodies   []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodiesMu.Lock()
		bodies = append(bodies, string(body))
		attempt := len(bodies)
		bodiesMu.Unlock()
		if attempt == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	restClient := newMiddlewareRestClient(t, server.URL)

	resp, err := restClient.Post("/v1/things", "application/json", strings.NewReader(`{"name":"thing"}`))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	resp.Body.Close()

	bodiesMu.Lock()
	defer bodiesMu.Unlock()
	if len(bodies) != 2 || bodies[0] != `{"name":"thing"}` || bodies[1] != `{"name":"thing"}` {
		t.Errorf("server received bodies %q, want the full body twice", bodies)
	}
}

type unreplayableReader struct{ reader io.Reader }

func (r *unreplayableReader) Read(p []byte) (int, error) { return r.reader.Read(p) }

func TestRetrySkippedWhenRequestBodyCannotBeReplayed(t *testing.T) {
	discardLogOutput(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	restClient := newMiddlewareRestClient(t, server.URL)

	_, err := restClient.Post("/v1/things", "application/json", &unreplayableReader{reader: strings.NewReader(`{"name":"thing"}`)})
	if !rest.HasStatusCode(err, http.StatusTooManyRequests) {
		t.Errorf("err = %v, want the original 429 response", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Errorf("server saw %d requests, want 1 because the body cannot be resent", got)
	}
}

func stringPointer(value string) *string { return &value }
