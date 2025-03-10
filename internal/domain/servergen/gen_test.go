package servergen //nolint:testpackage

import (
	"reflect"
	"testing"
)

func TestGetProtodirs(t *testing.T) {
	type args struct {
		protoPath string
		imports   []string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "deduced",
			args: args{
				protoPath: "protogen/example/multi-package/hello.proto",
				imports:   []string{"/protobuf"},
			},
			want: []string{
				"protogen/example/multi-package",
				"/protobuf",
			},
		},
		{
			name: "specified in imports",
			args: args{
				protoPath: "protogen/example/multi-package/hello.proto",
				imports:   []string{"/protobuf", "/example/multi-package/"},
			},
			want: []string{
				"protogen/example/multi-package",
				"/protobuf",
			},
		},
		{
			name: "specified in imports 2",
			args: args{
				protoPath: "protogen/example/multi-package/bar/bar.proto",
				imports:   []string{"example/multi-package", "/protobuf"},
			},
			want: []string{
				"protogen/example/multi-package",
				"/protobuf",
			},
		},
	}

	ctx := t.Context()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getProtodirs(ctx, tt.args.protoPath, tt.args.imports); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getProtodirs() = %v, want %v", got, tt.want)
			}
		})
	}
}
