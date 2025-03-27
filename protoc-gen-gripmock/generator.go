package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/imports"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	// Tip of the hat to Tim Coulson
	// https://medium.com/@tim.r.coulson/writing-a-protoc-plugin-with-google-golang-org-protobuf-cd5aa75f5777

	// Protoc passes pluginpb.CodeGeneratorRequest in via stdin
	// marshalled with Protobuf
	input, _ := io.ReadAll(os.Stdin)
	var request pluginpb.CodeGeneratorRequest
	if err := proto.Unmarshal(input, &request); err != nil {
		log.Fatalf("error unmarshalling [%s]: %v", string(input), err)
	}

	// Initialise our plugin with default options
	opts := protogen.Options{}
	plugin, err := opts.New(&request)
	if err != nil {
		log.Fatalf("error initializing plugin: %v", err)
	}

	plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

	protos := make([]*descriptorpb.FileDescriptorProto, len(plugin.Files))
	for index, file := range plugin.Files {
		protos[index] = file.Proto
	}

	params := make(map[string]string)
	for _, param := range strings.Split(request.GetParameter(), ",") {
		split := strings.Split(param, "=")
		params[split[0]] = split[1]
	}

	otlpTLS, _ := strconv.ParseBool(params["otlp-tls"])
	otlpRatioFloat, _ := strconv.ParseFloat(params["otlp-ratio"], 64)

	buf := new(bytes.Buffer)
	err = generateServer(protos, &Options{
		writer:          buf,
		adminHost:       params["admin-host"],
		adminPort:       params["admin-port"],
		grpcNet:         params["grpc-network"],
		grpcAddr:        net.JoinHostPort(params["grpc-address"], params["grpc-port"]),
		otlpTLS:         otlpTLS,
		otlpHost:        params["otlp-host"],
		otlpPort:        params["otlp-port"],
		otlpSampleRatio: otlpRatioFloat,
	})

	if err != nil {
		log.Fatalf("Failed to generate server %v", err)
	}

	file := plugin.NewGeneratedFile("server.go", ".")
	file.Write(buf.Bytes())

	// Generate a response from our plugin and marshall as protobuf
	out, err := proto.Marshal(plugin.Response())
	if err != nil {
		log.Fatalf("error marshalling plugin response: %v", err)
	}

	// Write the response to stdout, to be picked up by protoc
	os.Stdout.Write(out)
}

type generatorParam struct {
	Services        []Service
	Dependencies    map[string]string
	GrpcNet         string
	GrpcAddr        string
	AdminHost       string
	AdminPort       string
	OtlpHost        string
	OtlpPort        string
	OtlpTLS         bool
	OtlpSampleRatio float64
	PbPath          string
}

type Service struct {
	Name string
	// Name of the rpc handler struct. The name will be concatenation of RPC service name & package
	StructName string `json:"struct_name"`
	Package    string
	Methods    []methodTemplate
}

type methodTemplate struct {
	// Name is the name of the service.
	StructName  string `json:"struct_name"`
	SvcPackage  string
	Name        string
	ServiceName string
	MethodType  string
	Input       string
	Output      string
}

const (
	methodTypeStandard = "standard"
	// server to client stream
	methodTypeServerStream = "server-stream"
	// client to server stream
	methodTypeClientStream  = "client-stream"
	methodTypeBidirectional = "bidirectional"
)

type Options struct {
	writer          io.Writer
	grpcNet         string
	grpcAddr        string
	adminHost       string
	adminPort       string
	otlpHost        string
	otlpPort        string
	otlpTLS         bool
	otlpSampleRatio float64
	pbPath          string
}

var ServerTemplate string

//go:embed server.tmpl
var serverTmpl embed.FS

func init() {
	data, err := serverTmpl.ReadFile("server.tmpl")
	if err != nil {
		log.Fatalf("error reading server.tmpl: %s", err)
	}

	ServerTemplate = string(data)
}

func generateServer(protos []*descriptorpb.FileDescriptorProto, opt *Options) error {
	services := extractServices(protos)
	deps := resolveDependencies(protos)

	param := generatorParam{
		Services:        services,
		Dependencies:    deps,
		GrpcNet:         opt.grpcNet,
		GrpcAddr:        opt.grpcAddr,
		AdminHost:       opt.adminHost,
		AdminPort:       opt.adminPort,
		OtlpHost:        opt.otlpHost,
		OtlpPort:        opt.otlpPort,
		OtlpTLS:         opt.otlpTLS,
		OtlpSampleRatio: opt.otlpSampleRatio,
		PbPath:          opt.pbPath,
	}

	if opt == nil {
		opt = &Options{}
	}

	if opt.writer == nil {
		opt.writer = os.Stdout
	}

	tmpl := template.New("server.tmpl")
	tmpl, err := tmpl.Parse(ServerTemplate)
	if err != nil {
		return fmt.Errorf("template parse %v", err)
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, param)
	if err != nil {
		return fmt.Errorf("template execute %v", err)
	}

	byt := buf.Bytes()
	bytProcessed, err := imports.Process("", byt, nil)
	if err != nil {
		return fmt.Errorf("formatting: %v \n%s", err, string(byt))
	}

	_, err = opt.writer.Write(bytProcessed)
	return err
}

func resolveDependencies(protos []*descriptorpb.FileDescriptorProto) map[string]string {
	deps := map[string]string{}
	for _, proto := range protos {
		alias, pkg := getGoPackage(proto)

		// fatal if go_package is not present
		if pkg == "" {
			log.Fatalf("option go_package is required. but %s doesn't have any", proto.GetName())
		}

		if _, ok := deps[pkg]; ok {
			continue
		}

		deps[pkg] = alias
	}

	return deps
}

var (
	aliases  = map[string]bool{}
	aliasNum = 1
	packages = map[string]string{}
)

func getGoPackage(proto *descriptorpb.FileDescriptorProto) (alias string, goPackage string) {
	goPackage = proto.GetOptions().GetGoPackage()
	if goPackage == "" {
		return
	}

	// support go_package alias declaration
	// https://github.com/golang/protobuf/issues/139
	if splits := strings.Split(goPackage, ";"); len(splits) > 1 {
		goPackage = splits[0]
		alias = splits[1]
	} else {
		// get the alias based on the latest folder
		splitSlash := strings.Split(goPackage, "/")
		// replace - with _
		alias = strings.ReplaceAll(splitSlash[len(splitSlash)-1], "-", "_")
	}

	// if package already discovered just return
	if al, ok := packages[goPackage]; ok {
		alias = al
		return
	}

	// Aliases can't be keywords
	if isKeyword(alias) {
		alias = fmt.Sprintf("%s_pb", alias)
	}

	// in case of found same alias
	// add numbers on it
	if ok := aliases[alias]; ok {
		alias = fmt.Sprintf("%s%d", alias, aliasNum)
		aliasNum++
	}

	packages[goPackage] = alias
	aliases[alias] = true

	return
}

// change the structure also translate method type
func extractServices(protos []*descriptorpb.FileDescriptorProto) []Service {
	var svcTmp []Service
	title := cases.Title(language.English, cases.NoLower)
	for _, proto := range protos {
		for _, svc := range proto.GetService() {
			var s Service
			s.Name = svc.GetName()
			s.StructName = svc.GetName()
			alias, _ := getGoPackage(proto)
			if alias != "" {
				s.Package = alias + "."
				s.StructName = lo.PascalCase(alias) + s.Name
			}
			methods := make([]methodTemplate, len(svc.Method))
			for j, method := range svc.Method {
				tipe := methodTypeStandard
				if method.GetServerStreaming() && !method.GetClientStreaming() {
					tipe = methodTypeServerStream
				} else if !method.GetServerStreaming() && method.GetClientStreaming() {
					tipe = methodTypeClientStream
				} else if method.GetServerStreaming() && method.GetClientStreaming() {
					tipe = methodTypeBidirectional
				}

				methods[j] = methodTemplate{
					StructName:  s.StructName,
					Name:        title.String(*method.Name),
					SvcPackage:  s.Package,
					ServiceName: svc.GetName(),
					Input:       getMessageType(protos, method.GetInputType()),
					Output:      getMessageType(protos, method.GetOutputType()),
					MethodType:  tipe,
				}
			}
			s.Methods = methods
			svcTmp = append(svcTmp, s)
		}
	}
	return svcTmp
}

func getMessageType(protos []*descriptorpb.FileDescriptorProto, tipe string) string {
	split := strings.Split(tipe, ".")[1:]
	targetPackage := strings.Join(split[:len(split)-1], ".")
	targetType := split[len(split)-1]
	for _, proto := range protos {
		if proto.GetPackage() != targetPackage {
			continue
		}

		for _, msg := range proto.GetMessageType() {
			if msg.GetName() == targetType {
				alias, _ := getGoPackage(proto)
				if alias != "" {
					alias += "."
				}
				return fmt.Sprintf("%s%s", alias, msg.GetName())
			}
		}
	}
	return targetType
}

func isKeyword(word string) bool {
	keywords := [...]string{
		"break",
		"case",
		"chan",
		"const",
		"continue",
		"default",
		"defer",
		"else",
		"fallthrough",
		"for",
		"func",
		"go",
		"goto",
		"if",
		"import",
		"interface",
		"map",
		"package",
		"range",
		"return",
		"select",
		"struct",
		"switch",
		"type",
		"var",
	}

	for _, keyword := range keywords {
		if strings.ToLower(word) == keyword {
			return true
		}
	}

	return false
}
