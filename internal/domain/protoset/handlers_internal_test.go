package protoset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestParseSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		raw        string
		wantType   SourceType
		wantErr    bool
		wantPath   string
		wantModule string
		wantVer    string
	}{
		{
			name:       "buf.build without version",
			raw:        "buf.build/myorg/myservice",
			wantType:   SourceBufBuild,
			wantErr:    false,
			wantModule: "buf.build/myorg/myservice",
			wantVer:    "",
		},
		{
			name:       "buf.build with version",
			raw:        "buf.build/myorg/myservice:v1.0.0",
			wantType:   SourceBufBuild,
			wantErr:    false,
			wantModule: "buf.build/myorg/myservice",
			wantVer:    "v1.0.0",
		},
		{
			name:       "buf.build with digest",
			raw:        "buf.build/myorg/myservice@abc123def",
			wantType:   SourceBufBuild,
			wantErr:    false,
			wantModule: "buf.build/myorg/myservice",
			wantVer:    "abc123def",
		},
		{
			name:       "buf.build with branch",
			raw:        "buf.build/myorg/myservice:main",
			wantType:   SourceBufBuild,
			wantErr:    false,
			wantModule: "buf.build/myorg/myservice",
			wantVer:    "main",
		},
		{
			name:       "on-prem host without ref",
			raw:        "bsr.company.local/team/payments",
			wantType:   SourceBufBuild,
			wantErr:    false,
			wantModule: "bsr.company.local/team/payments",
			wantVer:    "",
		},
		{
			name:       "on-prem host with ref",
			raw:        "bsr.company.local/team/payments:main",
			wantType:   SourceBufBuild,
			wantErr:    false,
			wantModule: "bsr.company.local/team/payments",
			wantVer:    "main",
		},
		{
			name:     ".proto file",
			raw:      "service.proto",
			wantType: SourceProto,
			wantErr:  false,
			wantPath: "service.proto",
		},
		{
			name:     ".pb file",
			raw:      "service.pb",
			wantType: SourceDescriptor,
			wantErr:  false,
			wantPath: "service.pb",
		},
		{
			name:     ".protoset file",
			raw:      "service.protoset",
			wantType: SourceDescriptor,
			wantErr:  false,
			wantPath: "service.protoset",
		},
		{
			name:     "fallback to proto",
			raw:      "invalid",
			wantType: SourceProto,
			wantErr:  false,
			wantPath: "invalid",
		},
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

		require.True(t, h.CanHandle("buf.build/test/module"))
		require.True(t, h.CanHandle("bsr.company.local/team/module"))
		require.False(t, h.CanHandle("test.proto"))
		require.False(t, h.CanHandle("test.pb"))
		require.False(t, h.CanHandle("team/module/name"))

		src, err := h.Parse("buf.build/myorg/myservice:v1.0.0")
		require.NoError(t, err)
		require.Equal(t, SourceBufBuild, src.Type)
		require.Equal(t, "buf.build/myorg/myservice", src.Module)
		require.Equal(t, "v1.0.0", src.Version)
	})

	t.Run("DescriptorHandler", func(t *testing.T) {
		t.Parallel()

		h := &DescriptorHandler{}

		require.True(t, h.CanHandle("test.pb"))
		require.True(t, h.CanHandle("test.protoset"))
		require.False(t, h.CanHandle("test.proto"))
		require.False(t, h.CanHandle("buf.build/test/module"))

		src, err := h.Parse("test.pb")
		require.NoError(t, err)
		require.Equal(t, SourceDescriptor, src.Type)
		require.Equal(t, "test.pb", src.Path)
	})

	t.Run("ProtoHandler", func(t *testing.T) {
		t.Parallel()

		h := &ProtoHandler{}

		require.True(t, h.CanHandle("test.proto"))
		require.False(t, h.CanHandle("test.pb"))
		require.False(t, h.CanHandle("buf.build/test/module"))

		src, err := h.Parse("test.proto")
		require.NoError(t, err)
		require.Equal(t, SourceProto, src.Type)
		require.Equal(t, "test.proto", src.Path)
	})
}
