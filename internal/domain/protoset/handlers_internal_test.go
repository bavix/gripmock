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

	tests := makeParseSourceCases()

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
			case SourceReflect:
				require.NotEmpty(t, src.ReflectAddress)
			case SourceProto, SourceDescriptor:
				require.Equal(t, tt.wantPath, src.Path)
			case SourceDirectory, SourceUnknown:
			}
		})
	}
}

func makeParseSourceCases() []parseSourceCase {
	bsrInputs := []struct {
		name   string
		raw    string
		module string
		ver    string
	}{
		{name: "buf.build without version", raw: "buf.build/myorg/myservice", module: "buf.build/myorg/myservice"},
		{name: "buf.build with version", raw: "buf.build/myorg/myservice:v1.0.0", module: "buf.build/myorg/myservice", ver: "v1.0.0"},
		{name: "buf.build with digest", raw: "buf.build/myorg/myservice@abc123def", module: "buf.build/myorg/myservice", ver: "abc123def"},
		{name: "buf.build with branch", raw: "buf.build/myorg/myservice:main", module: "buf.build/myorg/myservice", ver: "main"},
		{name: "on-prem host without ref", raw: "bsr.company.local/team/payments", module: "bsr.company.local/team/payments"},
		{name: "on-prem host with ref", raw: "bsr.company.local/team/payments:main", module: "bsr.company.local/team/payments", ver: "main"},
	}

	cases := make([]parseSourceCase, 0, len(bsrInputs)+6)
	for _, in := range bsrInputs {
		cases = append(cases, bsrCase(in.name, in.raw, in.module, in.ver))
	}

	cases = append(cases,
		parseSourceCase{name: "grpc reflection source", raw: "grpc://localhost:50051", wantType: SourceReflect},
		parseSourceCase{name: "grpcs reflection source", raw: "grpcs://api.company.local:443", wantType: SourceReflect},
		fileCase(".proto file", "service.proto", SourceProto),
		fileCase(".pb file", "service.pb", SourceDescriptor),
		fileCase(".protoset file", "service.protoset", SourceDescriptor),
		fileCase("fallback to proto", "invalid", SourceProto),
	)

	return cases
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
