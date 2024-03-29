package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/imports"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

type Config struct {
	OTLPTrace OTLPTrace
	GRPC      GRPC
	HTTP      HTTP
}

type HTTP struct {
	Host string `envconfig:"HTTP_HOST" default:"0.0.0.0"`
	Port string `envconfig:"HTTP_PORT" default:"4771"`
}

type GRPC struct {
	Network string `envconfig:"GRPC_NETWORK" default:"tcp"`
	Host    string `envconfig:"GRPC_HOST" default:"0.0.0.0"`
	Port    string `envconfig:"GRPC_PORT" default:"4770"`
}

func Load() (Config, error) {
	cnf := Config{} //nolint:exhaustruct

	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return cnf, errors.Wrap(err, "read .env file")
	}

	if err := envconfig.Process("", &cnf); err != nil {
		return cnf, errors.Wrap(err, "read environment")
	}

	return cnf, nil
}

func (c *Config) GRPCAddr() string {
	return net.JoinHostPort(c.GRPC.Host, c.GRPC.Port)
}

func (c *Config) HTTPAddr() string {
	return net.JoinHostPort(c.HTTP.Host, c.HTTP.Port)
}

type OTLPTrace struct {
	Host        string  `envconfig:"OTLP_TRACE_GRPC_HOST" default:"127.0.0.1"`
	Port        string  `envconfig:"OTLP_TRACE_GRPC_PORT" default:"4317"`
	TLS         bool    `envconfig:"OTLP_TRACE_TLS" default:"false"`
	SampleRatio float64 `envconfig:"OTLP_SAMPLE_RATIO"`
}

func (o *OTLPTrace) UseTrace() bool {
	return o.Host != "" && o.Port != "" && o.SampleRatio > 0
}

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

	conf, err := Load()
	if err != nil {
		log.Fatal(err)
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

	// request.GetParameter()

	buf := new(bytes.Buffer)
	err = generateServer(protos, &Options{
		writer: buf,
		conf:   conf,
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
}

type Service struct {
	Name    string
	Package string
	Methods []methodTemplate
}

type methodTemplate struct {
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
	writer io.Writer
	conf   Config
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
		GrpcNet:         opt.conf.GRPC.Network,
		GrpcAddr:        opt.conf.GRPCAddr(),
		AdminHost:       opt.conf.HTTP.Host,
		AdminPort:       opt.conf.HTTP.Port,
		OtlpHost:        opt.conf.OTLPTrace.Host,
		OtlpPort:        opt.conf.OTLPTrace.Port,
		OtlpTLS:         opt.conf.OTLPTrace.TLS,
		OtlpSampleRatio: opt.conf.OTLPTrace.SampleRatio,
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
			alias, _ := getGoPackage(proto)
			if alias != "" {
				s.Package = alias + "."
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
