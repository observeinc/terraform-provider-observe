package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/observeinc/terraform-provider-observe/client/oid"
)

// TODO: generate from OpenAPI spec
type ReferenceTableInput struct {
	Metadata       ReferenceTableMetadataInput `json:"metadata"`
	SourceFilePath string                      `json:"-"`
	Schema         []ReferenceTableSchemaInput `json:"schema"`
}

type ReferenceTableSchemaInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// All fields are nullable + omitempty to support PATCH semantics (excluding field means leave as is)
type ReferenceTableMetadataInput struct {
	Label       *string   `json:"label,omitempty"`
	Description *string   `json:"description,omitempty"`
	PrimaryKey  *[]string `json:"primaryKey,omitempty"`
	LabelField  *string   `json:"labelField,omitempty"`
}

type ReferenceTable struct {
	Id          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Checksum    string `json:"checksum"`
	DatasetId   string `json:"datasetId"`
}

type ReferenceTableListResponse struct {
	TotalCount      int              `json:"totalCount"`
	ReferenceTables []ReferenceTable `json:"referenceTables"`
}

func (r *ReferenceTable) Oid() oid.OID {
	return oid.OID{
		Id:   r.Id,
		Type: oid.TypeReferenceTable,
	}
}

var errSourceFileChangedSize = errors.New("source file changed size during upload")

// MultipartFileUpload is a multipart/form-data request body whose file part is
// streamed from disk, so large files are never held in memory. The multipart
// framing around the file is rendered up front, which makes the exact
// Content-Length known before sending.
type MultipartFileUpload struct {
	contentType       string
	sourceFilePath    string
	sourceFileSize    int64
	framingBeforeFile []byte
	framingAfterFile  []byte
}

func (u *MultipartFileUpload) ContentType() string {
	return u.contentType
}

func (u *MultipartFileUpload) ContentLength() int64 {
	return int64(len(u.framingBeforeFile)) + u.sourceFileSize + int64(len(u.framingAfterFile))
}

// Open returns a reader over the complete body. Each call re-opens the source
// file, so it can be used as http.Request.GetBody to resend the body.
func (u *MultipartFileUpload) Open() (io.ReadCloser, error) {
	file, err := os.Open(u.sourceFilePath)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if info.Size() != u.sourceFileSize {
		file.Close()
		return nil, fmt.Errorf("%w: %s was %d bytes, now %d", errSourceFileChangedSize, u.sourceFilePath, u.sourceFileSize, info.Size())
	}
	fileContent := &sizeCheckedFileReader{file: file, path: u.sourceFilePath, remainingBytes: u.sourceFileSize}
	return &multipartFileUploadReader{
		Reader: io.MultiReader(bytes.NewReader(u.framingBeforeFile), fileContent, bytes.NewReader(u.framingAfterFile)),
		file:   file,
	}, nil
}

type multipartFileUploadReader struct {
	io.Reader
	file *os.File
}

func (r *multipartFileUploadReader) Close() error {
	return r.file.Close()
}

// sizeCheckedFileReader yields exactly remainingBytes from file and fails if the
// file is shorter or longer, because the Content-Length has already been fixed.
type sizeCheckedFileReader struct {
	file           *os.File
	path           string
	remainingBytes int64
}

func (r *sizeCheckedFileReader) Read(p []byte) (int, error) {
	if r.remainingBytes == 0 {
		var probe [1]byte
		n, err := r.file.Read(probe[:])
		switch {
		case n > 0:
			return 0, fmt.Errorf("%w: %s grew", errSourceFileChangedSize, r.path)
		case err == nil || errors.Is(err, io.EOF):
			return 0, io.EOF
		default:
			return 0, err
		}
	}
	if int64(len(p)) > r.remainingBytes {
		p = p[:r.remainingBytes]
	}
	n, err := r.file.Read(p)
	r.remainingBytes -= int64(n)
	if errors.Is(err, io.EOF) {
		if r.remainingBytes > 0 {
			return n, fmt.Errorf("%w: %s shrank", errSourceFileChangedSize, r.path)
		}
		err = nil
	}
	return n, err
}

// redirectableWriter lets a single multipart.Writer render the framing before
// and after the file part into separate buffers.
type redirectableWriter struct {
	target io.Writer
}

func (w *redirectableWriter) Write(p []byte) (int, error) {
	return w.target.Write(p)
}

func (r *ReferenceTableInput) RequestBody() (*MultipartFileUpload, error) {
	info, err := os.Stat(r.SourceFilePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source file %s is not a regular file", r.SourceFilePath)
	}
	schema, err := json.Marshal(r.Schema)
	if err != nil {
		return nil, err
	}
	metadata, err := json.Marshal(r.Metadata)
	if err != nil {
		return nil, err
	}

	var framingBeforeFile, framingAfterFile bytes.Buffer
	framingDestination := &redirectableWriter{target: &framingBeforeFile}
	writer := multipart.NewWriter(framingDestination)
	if _, err := writer.CreateFormFile("upload", filepath.Base(r.SourceFilePath)); err != nil {
		return nil, err
	}
	framingDestination.target = &framingAfterFile
	schemaPart, err := writer.CreateFormFile("schema", "schema.json")
	if err != nil {
		return nil, err
	}
	if _, err := schemaPart.Write(schema); err != nil {
		return nil, err
	}
	if err := writer.WriteField("metadata", string(metadata)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return &MultipartFileUpload{
		contentType:       writer.FormDataContentType(),
		sourceFilePath:    r.SourceFilePath,
		sourceFileSize:    info.Size(),
		framingBeforeFile: framingBeforeFile.Bytes(),
		framingAfterFile:  framingAfterFile.Bytes(),
	}, nil
}

func (client *Client) CreateReferenceTable(ctx context.Context, input *ReferenceTableInput) (*ReferenceTable, error) {
	upload, err := input.RequestBody()
	if err != nil {
		return nil, err
	}

	resp, err := client.requestWithUpload(ctx, http.MethodPost, "/v1/referencetables", upload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}
	return refTable, nil
}

func (client *Client) GetReferenceTable(ctx context.Context, id string) (*ReferenceTable, error) {
	resp, err := client.Get("/v1/referencetables/" + url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}

	return refTable, nil
}

func (client *Client) UpdateReferenceTable(ctx context.Context, id string, input *ReferenceTableInput) (*ReferenceTable, error) {
	upload, err := input.RequestBody()
	if err != nil {
		return nil, err
	}
	resp, err := client.requestWithUpload(ctx, http.MethodPut, "/v1/referencetables/"+url.PathEscape(id), upload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}
	return refTable, nil
}

func (client *Client) UpdateReferenceTableMetadata(ctx context.Context, id string, input *ReferenceTableMetadataInput) (*ReferenceTable, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch("/v1/referencetables/"+url.PathEscape(id), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTable := &ReferenceTable{}
	if err := json.NewDecoder(resp.Body).Decode(refTable); err != nil {
		return nil, err
	}
	return refTable, nil
}

func (client *Client) DeleteReferenceTable(ctx context.Context, id string) error {
	resp, err := client.Delete("/v1/referencetables/" + url.PathEscape(id))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (client *Client) LookupReferenceTable(ctx context.Context, label string) (*ReferenceTable, error) {
	resp, err := client.Get("/v1/referencetables?label=" + url.QueryEscape(label))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	refTableList := &ReferenceTableListResponse{}
	if err := json.NewDecoder(resp.Body).Decode(refTableList); err != nil {
		return nil, err
	}

	// the API does a substring match, we want an exact match
	var refTable *ReferenceTable
	for _, t := range refTableList.ReferenceTables {
		if t.Label == label {
			refTable = &t
			break
		}
	}

	if refTable == nil {
		return nil, fmt.Errorf("reference table not found")
	}

	return refTable, nil
}
