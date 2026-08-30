package protoset

import (
	"context"
	"slices"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type fileRegistration int

const (
	fileNotRegistered fileRegistration = iota
	fileRegisteredIdentical
	fileRegisteredDifferent
)

func registeredFileState(name string, candidate *descriptorpb.FileDescriptorProto) fileRegistration {
	existing, err := protoregistry.GlobalFiles.FindFileByPath(name)
	if err != nil || existing == nil {
		return fileNotRegistered
	}

	if candidate == nil || proto.Equal(protodesc.ToFileDescriptorProto(existing), candidate) {
		return fileRegisteredIdentical
	}

	return fileRegisteredDifferent
}

func logAlreadyRegistered(ctx context.Context, name, path string, state fileRegistration) {
	if state != fileRegisteredDifferent {
		zerolog.Ctx(ctx).Debug().
			Str("name", name).
			Str("path", path).
			Msg("file already registered with identical content; skipping")

		return
	}

	zerolog.Ctx(ctx).Warn().
		Str("name", name).
		Str("path", path).
		Msg("a different file is already registered under this name: its services will be served " +
			"and the ones defined here will be missing. Proto files are identified by the name they " +
			"were compiled under, so give the two files distinct names or distinct import paths")
}

//nolint:gochecknoglobals // shared lock is required for GlobalFiles concurrent registration safety
var protoRegistryMu sync.Mutex

var (
	errUnresolvedDescriptorDependencies = errors.New("unresolved descriptor dependencies")
	errDescriptorSymbolConflict         = errors.New("descriptor symbol conflict")
)

//nolint:funlen,wsl_v5
func RegisterDescriptorSetFiles(
	ctx context.Context,
	descriptorPath string,
	fds *descriptorpb.FileDescriptorSet,
) error {
	renameCollidingFiles(ctx, descriptorPath, fds)

	pending := slices.Clone(fds.GetFile())

	for len(pending) > 0 {
		next := make([]*descriptorpb.FileDescriptorProto, 0, len(pending))
		progress := false
		var lastErr error

		for _, fd := range pending {
			protoRegistryMu.Lock()

			if state := registeredFileState(fd.GetName(), fd); state != fileNotRegistered {
				protoRegistryMu.Unlock()

				logAlreadyRegistered(ctx, fd.GetName(), descriptorPath, state)

				progress = true

				continue
			}

			fileDesc, err := protodesc.NewFile(fd, protoregistry.GlobalFiles)
			if err != nil {
				protoRegistryMu.Unlock()
				lastErr = err
				next = append(next, fd)

				continue
			}

			conflict, registerErr := registerGlobalFile(fileDesc)
			protoRegistryMu.Unlock()

			if conflict {
				zerolog.Ctx(ctx).Warn().
					Str("name", fd.GetName()).
					Str("path", descriptorPath).
					Msg("Descriptor conflicts with existing symbols; skipping")

				progress = true

				continue
			}

			if registerErr != nil {
				lastErr = registerErr
				next = append(next, fd)

				continue
			}

			progress = true
		}

		if len(next) == 0 {
			return nil
		}

		if !progress {
			return errors.Wrapf(lastErr, "%w: failed to register file %s", errUnresolvedDescriptorDependencies, descriptorPath)
		}

		pending = next
	}

	return nil
}

func registerGlobalFile(file protoreflect.FileDescriptor) (bool, error) {
	var (
		conflict bool
		err      error
	)

	defer func() {
		if recovered := recover(); recovered != nil {
			conflict = true
			err = errors.Wrapf(errDescriptorSymbolConflict, "%v", recovered)
		}
	}()

	err = protoregistry.GlobalFiles.RegisterFile(file)

	return conflict, err
}

func registerGlobalFileOnce(
	ctx context.Context,
	fileName string,
	filePath string,
	file protoreflect.FileDescriptor,
) error {
	protoRegistryMu.Lock()
	defer protoRegistryMu.Unlock()

	if state := registeredFileState(fileName, protodesc.ToFileDescriptorProto(file)); state != fileNotRegistered {
		logAlreadyRegistered(ctx, fileName, filePath, state)

		return nil
	}

	conflict, err := registerGlobalFile(file)
	if conflict {
		zerolog.Ctx(ctx).Warn().
			Str("name", fileName).
			Str("path", filePath).
			Msg("Descriptor conflicts with existing symbols; skipping")

		return nil
	}

	if err != nil {
		return errors.Wrapf(err, "failed to register file %s", filePath)
	}

	return nil
}
