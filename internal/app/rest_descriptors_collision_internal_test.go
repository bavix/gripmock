package app

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestUploadedDescriptorDoesNotEvictAnEarlierOneWithTheSameFileName(t *testing.T) {
	t.Parallel()

	server, err := NewRestServer(t.Context(), stuber.NewBudgerigar(), nopExtender{}, nil, nil, nil, nil)
	require.NoError(t, err)

	first, err := registerDescriptorBytes(server, descriptorSetBytes(t, "collision.proto", "collision.first", "FirstService"))
	require.NoError(t, err)
	require.Equal(t, []string{"collision.first.FirstService"}, first)

	// Same file name, different content: the registry is keyed by path, so this
	// used to replace the first upload and silently drop its service.
	second, err := registerDescriptorBytes(server, descriptorSetBytes(t, "collision.proto", "collision.second", "SecondService"))
	require.NoError(t, err)
	require.Equal(t, []string{"collision.second.SecondService"}, second)

	require.ElementsMatch(t,
		[]string{"collision.first.FirstService", "collision.second.SecondService"},
		server.restDescriptors.ServiceIDs())
}

func descriptorSetBytes(t *testing.T, fileName, pkg, service string) []byte {
	t.Helper()

	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{{
			Name:    new(fileName),
			Package: new(pkg),
			Syntax:  new("proto3"),
			MessageType: []*descriptorpb.DescriptorProto{
				{Name: new("Req")},
				{Name: new("Res")},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: new(service),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:       new("Call"),
					InputType:  new("." + pkg + ".Req"),
					OutputType: new("." + pkg + ".Res"),
				}},
			}},
		}},
	}

	raw, err := proto.Marshal(fds)
	require.NoError(t, err)

	return raw
}
