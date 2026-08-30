package protoset

import (
	"context"
	"path"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// renameCollidingFiles gives a unique path to every file of the set whose name is
// already taken by a *different* file.
//
// Descriptor files are identified by the name they were compiled under, and
// "service.proto" is the most common name there is: two upstreams behind one
// GripMock routinely ship different files under it. Skipping the second one lost
// its services entirely, so instead it is re-registered under a name derived from
// its source. Symbols are untouched — a genuine symbol clash is still reported.
//
// The set is rewritten in place: dependencies inside it follow the rename, so the
// caller keeps working with names that match the registry.
func renameCollidingFiles(ctx context.Context, descriptorPath string, fds *descriptorpb.FileDescriptorSet) {
	files := fds.GetFile()
	if len(files) == 0 {
		return
	}

	protoRegistryMu.Lock()
	defer protoRegistryMu.Unlock()

	prefix := UniqueFilePrefix(descriptorPath)
	renames := make(map[string]string)

	for _, file := range files {
		if registeredFileState(file.GetName(), file) != fileRegisteredDifferent {
			continue
		}

		// A vendored copy of a well-known file (google/protobuf/descriptor.proto and
		// friends) differs from the compiled-in one byte for byte while declaring the
		// very same symbols. Renaming it would only move the clash: registration
		// rejects duplicate symbols whatever the file is called, and then every file
		// importing it fails to resolve. The registered copy already serves them.
		if symbolsAlreadyRegistered(file) {
			continue
		}

		renames[file.GetName()] = path.Join(prefix, file.GetName())
	}

	if len(renames) == 0 {
		return
	}

	ApplyFileRenames(files, renames)

	// Re-fetching the same upstream must land on the same name, so a file that is
	// already registered under the new name with identical content is left alone;
	// only a real clash there moves on to the next suffix.
	for _, file := range files {
		for attempt := 2; registeredFileState(file.GetName(), file) == fileRegisteredDifferent; attempt++ {
			next := path.Join(prefix+"-"+strconv.Itoa(attempt), path.Base(file.GetName()))
			ApplyFileRenames(files, map[string]string{file.GetName(): next})
		}
	}

	for original, renamed := range renames {
		zerolog.Ctx(ctx).Info().
			Str("name", original).
			Str("registered_as", renamed).
			Str("path", descriptorPath).
			Msg("file name already taken by a different descriptor; registering this one under a unique name")
	}
}

// symbolsAlreadyRegistered reports whether any top-level symbol of the file is taken
// in the global registry: such a file cannot be registered under any name.
func symbolsAlreadyRegistered(file *descriptorpb.FileDescriptorProto) bool {
	pkg := file.GetPackage()

	names := make([]string, 0, len(file.GetMessageType())+len(file.GetEnumType())+
		len(file.GetService())+len(file.GetExtension()))

	for _, message := range file.GetMessageType() {
		names = append(names, message.GetName())
	}

	for _, enum := range file.GetEnumType() {
		names = append(names, enum.GetName())
	}

	for _, service := range file.GetService() {
		names = append(names, service.GetName())
	}

	for _, extension := range file.GetExtension() {
		names = append(names, extension.GetName())
	}

	for _, name := range names {
		full := protoreflect.FullName(name)
		if pkg != "" {
			full = protoreflect.FullName(pkg + "." + name)
		}

		if _, err := protoregistry.GlobalFiles.FindDescriptorByName(full); err == nil {
			return true
		}
	}

	return false
}

// ApplyFileRenames rewrites file names and the dependencies pointing at them, so a
// renamed file keeps being referenced by the files that came with it.
func ApplyFileRenames(files []*descriptorpb.FileDescriptorProto, renames map[string]string) {
	for _, file := range files {
		if renamed, ok := renames[file.GetName()]; ok {
			file.Name = &renamed
		}

		for i, dependency := range file.GetDependency() {
			if renamed, ok := renames[dependency]; ok {
				file.Dependency[i] = renamed
			}
		}
	}
}

// UniqueFilePrefix builds the path segment a renamed file is placed under.
func UniqueFilePrefix(descriptorPath string) string {
	return path.Join("gripmock", sanitizeDescriptorPath(descriptorPath))
}

// sanitizeDescriptorPath turns a source label (a file path, a proxy address) into
// one path segment.
func sanitizeDescriptorPath(descriptorPath string) string {
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', ' ':
			return '_'
		default:
			return r
		}
	}, strings.TrimSpace(descriptorPath))

	if replaced == "" {
		return "source"
	}

	return replaced
}
