package protobundle

import (
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/klauspost/compress/s2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// pbsExt is the file extension for S2-compressed descriptor bundles.
const pbsExt = ".pbs"

// Write marshals the FileDescriptorSet and writes it atomically to path.
// If the path ends with ".pbs", the payload is compressed with S2 (block mode, best compression).
// Otherwise (".pb") the raw protobuf bytes are written.
// It writes to a temporary file first, then renames to ensure atomicity.
func Write(fds *descriptorpb.FileDescriptorSet, path string) error {
	data, err := proto.Marshal(fds)
	if err != nil {
		return errors.Wrap(err, "failed to marshal descriptor set")
	}

	if filepath.Ext(path) == pbsExt {
		data = s2.EncodeBest(nil, data)
	}

	return writeAtomic(path, data)
}

// Decode decompresses S2-compressed data and unmarshals a FileDescriptorSet.
// This is the counterpart of Write with ".pbs" extension, intended for use at load time.
func Decode(compressed []byte) (*descriptorpb.FileDescriptorSet, error) {
	raw, err := s2.Decode(nil, compressed)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decompress S2 data")
	}

	fds := &descriptorpb.FileDescriptorSet{}
	if err = proto.Unmarshal(raw, fds); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal descriptor set")
	}

	return fds, nil
}

// writeAtomic writes data to path atomically via a temporary file + rename.
func writeAtomic(path string, data []byte) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve output path: %s", path)
	}

	dir := filepath.Dir(absPath)

	if err = os.MkdirAll(dir, 0o750); err != nil { //nolint:mnd
		return errors.Wrapf(err, "failed to create output directory: %s", dir)
	}

	tmp := absPath + ".tmp"

	if err = os.WriteFile(tmp, data, 0o600); err != nil { //nolint:mnd
		return errors.Wrapf(err, "failed to write temporary file: %s", tmp)
	}

	if err = os.Rename(tmp, absPath); err != nil {
		_ = os.Remove(tmp) // best-effort cleanup

		return errors.Wrapf(err, "failed to rename %s to %s", tmp, absPath)
	}

	return nil
}
