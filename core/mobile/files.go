package mobile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// App-container evidence is immutable input to readers; reject symlink escapes,
// devices and oversized files before allocating or scanning their contents.
func openEvidence(directory, name string, limit int64) (*os.File, error) {
	path := filepath.Join(directory, name)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || resolved != path {
		return nil, fault("corrupt_data", "openEvidence", fmt.Errorf("%s is not a local regular file", name))
	}
	if info.Size() > limit {
		return nil, fault("resource_limit", "openEvidence", fmt.Errorf("%s exceeds %d bytes", name, limit))
	}
	return os.Open(path)
}
func readDocument(directory, name string) ([]byte, error) {
	file, err := openEvidence(directory, name, maxJSONBytes)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxJSONBytes+1))
	if len(data) > maxJSONBytes {
		return nil, fault("resource_limit", "readDocument", fmt.Errorf("document too large"))
	}
	return data, err
}
func decodeDocument(data []byte, target any) error {
	if len(data) > maxJSONBytes {
		return fault("resource_limit", "decodeDocument", fmt.Errorf("document too large"))
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fault("corrupt_data", "decodeDocument", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fault("corrupt_data", "decodeDocument", fmt.Errorf("expected one JSON document"))
	}
	return nil
}
