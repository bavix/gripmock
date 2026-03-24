package protoset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type parseSourceCase struct {
	name       string
	raw        string
	wantType   SourceType
	wantErr    bool
	wantPath   string
	wantModule string
	wantVer    string
}

func bsrCase(name, raw, module, ver string) parseSourceCase {
	return parseSourceCase{
		name:       name,
		raw:        raw,
		wantType:   SourceBufBuild,
		wantErr:    false,
		wantModule: module,
		wantVer:    ver,
	}
}

func fileCase(name, raw string, typ SourceType) parseSourceCase {
	return parseSourceCase{
		name:     name,
		raw:      raw,
		wantType: typ,
		wantErr:  false,
		wantPath: raw,
	}
}

func assertCanHandle(t *testing.T, h SourceHandler, valid []string, invalid []string) {
	t.Helper()

	for _, raw := range valid {
		require.True(t, h.CanHandle(raw))
	}

	for _, raw := range invalid {
		require.False(t, h.CanHandle(raw))
	}
}

func TestParseSource(t *testing.T) {
	t.Parallel()

	tests := []parseSourceCase{
		bsrCase("buf.build without version", "buf.build/myorg/myservice", "buf.build/myorg/myservice", ""),
		bsrCase("buf.build with version", "buf.build/myorg/myservice:v1.0.0", "buf.build/myorg/myservice", "v1.0.0"),
		bsrCase("buf.build with digest", "buf.build/myorg/myservice@abc123def", "buf.build/myorg/myservice", "abc123def"),
		bsrCase("buf.build with branch", "buf.build/myorg/myservice:main", "buf.build/myorg/myservice", "main"),
		bsrCase("on-prem host without ref", "bsr.company.local/team/payments", "bsr.company.local/team/payments", ""),
		bsrCase("on-prem host with ref", "bsr.company.local/team/payments:main", "bsr.company.local/team/payments", "main"),
		fileCase(".proto file", "service.proto", SourceProto),
		fileCase(".pb file", "service.pb", SourceDescriptor),
		fileCase(".protoset file", "service.protoset", SourceDescriptor),
		fileCase("fallback to proto", "invalid", SourceProto),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src, err := ParseSource(tt.raw)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantType, src.Type)
			require.Equal(t, tt.raw, src.Raw)

			switch tt.wantType {
			case SourceBufBuild:
				require.Equal(t, tt.wantModule, src.Module)
				require.Equal(t, tt.wantVer, src.Version)
			case SourceProto, SourceDescriptor:
				require.Equal(t, tt.wantPath, src.Path)
			case SourceDirectory, SourceUnknown:
			}
		})
	}
}

func TestHandlers(t *testing.T) {
	t.Parallel()

	t.Run("BufBuildHandler", func(t *testing.T) {
		t.Parallel()

		h := &BufBuildHandler{}
		assertCanHandle(t, h,
			[]string{"buf.build/test/module", "bsr.company.local/team/module"},
			[]string{"test.proto", "test.pb", "team/module/name"},
		)

		src, err := h.Parse("buf.build/myorg/myservice:v1.0.0")
		require.NoError(t, err)
		require.Equal(t, SourceBufBuild, src.Type)
		require.Equal(t, "buf.build/myorg/myservice", src.Module)
		require.Equal(t, "v1.0.0", src.Version)
	})

	t.Run("DescriptorHandler", func(t *testing.T) {
		t.Parallel()

		h := &DescriptorHandler{}
		assertCanHandle(t, h,
			[]string{"test.pb", "test.protoset"},
			[]string{"test.proto", "buf.build/test/module"},
		)

		src, err := h.Parse("test.pb")
		require.NoError(t, err)
		require.Equal(t, SourceDescriptor, src.Type)
		require.Equal(t, "test.pb", src.Path)
	})

	t.Run("ProtoHandler", func(t *testing.T) {
		t.Parallel()

		h := &ProtoHandler{}
		assertCanHandle(t, h,
			[]string{"test.proto"},
			[]string{"test.pb", "buf.build/test/module"},
		)

		src, err := h.Parse("test.proto")
		require.NoError(t, err)
		require.Equal(t, SourceProto, src.Type)
		require.Equal(t, "test.proto", src.Path)
	})
}
