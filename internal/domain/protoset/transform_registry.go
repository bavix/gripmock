package protoset

import (
	"context"
	"slices"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

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
	pending := slices.Clone(fds.GetFile())

	for len(pending) > 0 {
		next := make([]*descriptorpb.FileDescriptorProto, 0, len(pending))
		progress := false
		var lastErr error

		for _, fd := range pending {
			protoRegistryMu.Lock()

			if value, _ := protoregistry.GlobalFiles.FindFileByPath(fd.GetName()); value != nil {
				protoRegistryMu.Unlock()

				zerolog.Ctx(ctx).Warn().
					Str("name", fd.GetName()).
					Str("path", descriptorPath).
					Msg("File already registered")

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

	if value, _ := protoregistry.GlobalFiles.FindFileByPath(fileName); value != nil {
		zerolog.Ctx(ctx).Warn().
			Str("name", fileName).
			Str("path", filePath).
			Msg("File already registered")

		return nil
	}

	err := protoregistry.GlobalFiles.RegisterFile(file)
	if err != nil {
		return errors.Wrapf(err, "failed to register file %s", filePath)
	}

	return nil
}
