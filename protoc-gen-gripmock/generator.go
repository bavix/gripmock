// Package main contains the main function for the protoc-gen-gripmock generator.
package main // import "github.com/bavix/gripmock/protoc-gen-gripmock"

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/imports"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// This package contains the implementation of protoc-gen-gripmock.
// It uses the protoc tool to generate a gRPC mock server from a .proto file.
//
// This package is generated by the go generate tag and should not be edited
// by hand.

// The main function is the entry point for the protoc-gen-gripmock generator.
// It reads input from stdin, unmarshals the request, and creates a new
// CodeGenerator object. It then generates the gRPC mock server in server.go.
func main() {

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

	// Create a slice of FileDescriptorProto objects for each input file.
	protos := make([]*descriptorpb.FileDescriptorProto, len(plugin.Files))
	for index, file := range plugin.Files {
		protos[index] = file.Proto
	}

	// Generate the gRPC mock server using the input files.
	buf := new(bytes.Buffer)
	err = generateServer(protos, &Options{
		Writer: buf,
	})
	if err != nil {
		log.Fatalf("Failed to generate server %v", err)
	}

	// Create a new GeneratedFile with the name "server.go" and ".go" extension.
	file := plugin.NewGeneratedFile("server.go", ".go")

	// Write the generated gRPC mock server code to the GeneratedFile.
	file.Write(buf.Bytes())

	// Generate a response from our plugin and marshall as protobuf
	out, err := proto.Marshal(plugin.Response())
	if err != nil {
		log.Fatalf("error marshalling plugin response: %v", err)
	}

	// Write the response to stdout, to be picked up by protoc
	os.Stdout.Write(out)
}

// generatorParam contains the parameters used to generate the gRPC mock server.
type generatorParam struct {
	// Services is a slice of Service objects representing the services and methods in the input files.
	Services []Service `json:"services"`
	// Dependencies is a map of package names to their respective import paths.
	// It is used to generate the import statements at the top of the generated server file.
	Dependencies map[string]string `json:"dependencies"`
}

// Service represents a gRPC service.
type Service struct {
	// Name is the name of the service.
	Name string `json:"name"`
	// Package is the package name of the service.
	Package string `json:"package"`
	// Methods is a slice of methodTemplate representing the methods in the service.
	Methods []methodTemplate `json:"methods"`
}

// methodTemplate represents a method in a gRPC service.
type methodTemplate struct {
	// SvcPackage is the package name of the service.
	SvcPackage string `json:"svc_package"`
	// Name is the name of the method.
	Name string `json:"name"`
	// ServiceName is the name of the service.
	ServiceName string `json:"service_name"`
	// MethodType is the type of the method, which can be "standard", "server-stream", "client-stream", or "bidirectional".
	MethodType string `json:"method_type"`
	// Input is the name of the input message for the method.
	Input string `json:"input"`
	// Output is the name of the output message for the method.
	Output string `json:"output"`
}

const (
	methodTypeStandard = "standard"
	// server to client stream
	// methodTypeServerStream represents a server-stream method.
	methodTypeServerStream = "server-stream"
	// methodTypeClientStream represents a client-stream method.
	methodTypeClientStream = "client-stream"
	// methodTypeBidirectional represents a bidirectional method.
	methodTypeBidirectional = "bidirectional"
)

// Options holds the configuration options for the code generator.
type Options struct {
	// Writer is the io.Writer used to write the generated server code.
	// If not provided, the generated code is written to stdout.
	Writer io.Writer `json:"writer"`
}

// ServerTemplate is the template used to generate the gRPC server code.
// It is populated during the init function.
var ServerTemplate string

// serverTmpl is the embed.FS used to read the server template file.
//
//go:embed server.tmpl
var serverTmpl embed.FS

// Init initializes the ServerTemplate with the contents of the server.tmpl file.
//
// It reads the server.tmpl file from the serverTmpl embed.FS and assigns its contents
// to the ServerTemplate variable. If there is an error reading the file, it logs
// the error and stops the program.
func init() {
	data, err := serverTmpl.ReadFile("server.tmpl")
	if err != nil {
		log.Fatalf("error reading server.tmpl: %s", err)
	}

	ServerTemplate = string(data)
}

// generateServer generates the gRPC server code based on the given protobuf
// descriptors and writes it to the provided io.Writer.
//
// It extracts the services from the given protobuf descriptors, resolves their
// dependencies, and generates the server code using the provided options.
// If no io.Writer is provided in the options, the generated code is written to
// os.Stdout.
//
// It returns an error if there is any issue in generating or writing the code.
func generateServer(protos []*descriptorpb.FileDescriptorProto, opt *Options) error {
	// Extract the services from the given protobuf descriptors
	services := extractServices(protos)

	// Resolve the dependencies of the services
	deps := resolveDependencies(protos)

	// Prepare the parameters for generating the server code
	param := generatorParam{
		Services:     services,
		Dependencies: deps,
	}

	// If no io.Writer is provided in the options, use os.Stdout
	if opt.Writer == nil {
		opt.Writer = os.Stdout
	}

	// Create a new template and parse the server template
	tmpl := template.New("server.tmpl")
	_, err := tmpl.Parse(ServerTemplate)
	if err != nil {
		return fmt.Errorf("template parse %v", err)
	}

	// Execute the template with the parameters and write the generated code to a buffer
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, param)
	if err != nil {
		return fmt.Errorf("template execute %v", err)
	}

	// Format the generated code using gofmt
	byt := buf.Bytes()
	bytProcessed, err := imports.Process("", byt, nil)
	if err != nil {
		return fmt.Errorf("formatting: %v \n%s", err, string(byt))
	}

	// Write the formatted code to the io.Writer
	_, err = opt.Writer.Write(bytProcessed)
	return err
}

// resolveDependencies takes a list of protobuf file descriptors and returns a
// map of go package names to their respective alias. It resolves the
// dependencies by checking the go_package option of each protobuf file. If a
// go_package option is not present, it logs a fatal error. If a go package
// already exists in the map, it is skipped.
//
// Parameters:
// - protos: a list of protobuf file descriptors
//
// Returns:
// - a map of go package names to their respective alias
func resolveDependencies(protos []*descriptorpb.FileDescriptorProto) map[string]string {
	deps := map[string]string{}

	// Iterate over each protobuf file descriptor
	for _, proto := range protos {
		// Get the go package alias and name from the protobuf file descriptor
		alias, pkg := getGoPackage(proto)

		// Log a fatal error if the go_package option is not present
		if pkg == "" {
			log.Fatalf("option go_package is required. but %s doesn't have any", proto.GetName())
		}

		// Skip the go package if it already exists in the map
		if _, ok := deps[pkg]; ok {
			continue
		}

		// Add the go package to the map with its alias
		deps[pkg] = alias
	}

	return deps
}

var (
	// aliases is a map that keeps track of package aliases. The key is the alias and the value
	// is a boolean indicating whether the alias is used or not.
	aliases = map[string]bool{}

	// aliasNum is an integer that keeps track of the number of used aliases. It is used to
	// generate new unique aliases.
	aliasNum = 1

	// packages is a map that stores the package names as keys and their corresponding aliases
	// as values. The package names are the full go package names and the aliases are the
	// generated or specified aliases for the packages.
	packages = map[string]string{}
)

// getGoPackage returns the go package alias and the go package name
// extracted from the protobuf file's go_package option.
//
// If the go_package option is not present, it returns an empty string for goPackage.
// If the go_package option is present and has no alias, it returns an empty string for alias.
//
// If the go_package option has an alias, it returns the alias and the go package name.
// The alias is derived from the last folder in the go package name.
// If the last folder contains a dash, it is replaced with an underscore.
// If the alias is a keyword, it appends a random number to the alias.
// If the alias already exists, it appends a number to the alias.
func getGoPackage(proto *descriptorpb.FileDescriptorProto) (alias string, goPackage string) {
	goPackage = proto.GetOptions().GetGoPackage()
	if goPackage == "" {
		return
	}

	// Support go_package alias declaration
	// https://github.com/golang/protobuf/issues/139
	if splits := strings.Split(goPackage, ";"); len(splits) > 1 {
		goPackage = splits[0]
		alias = splits[1]
	} else {
		// Get the alias based on the last folder in the go package name
		splitSlash := strings.Split(goPackage, "/")
		// Replace dash with underscore
		alias = strings.ReplaceAll(splitSlash[len(splitSlash)-1], "-", "_")
	}

	// If the package has already been discovered, return the alias
	if al, ok := packages[goPackage]; ok {
		alias = al
		return
	}

	// Aliases can't be keywords
	if isKeyword(alias) {
		// Append a random number to the alias
		alias = fmt.Sprintf("%s_%x_pb", alias, rand.Int())
	}

	// If the alias already exists, append a number to the alias
	if ok := aliases[alias]; ok {
		alias = fmt.Sprintf("%s%d", alias, aliasNum)
		aliasNum++
	}

	packages[goPackage] = alias
	aliases[alias] = true

	return
}

// extractServices extracts services from a list of file descriptors. It returns
// a slice of Service structs, each representing a gRPC service.
//
// The function iterates over each file descriptor and extracts the services
// defined in each file. It then populates the Services struct with relevant
// information like the service name, package name, and methods. The methods
// include information such as the method name, input and output types, and the
// type of method (standard, server-stream, client-stream, or bidirectional).
//
// Parameters:
// - protos: A slice of FileDescriptorProto structs representing the file
// descriptors.
//
// Returns:
// - svcTmp: A slice of Service structs representing the extracted services.
func extractServices(protos []*descriptorpb.FileDescriptorProto) []Service {
	var svcTmp []Service
	title := cases.Title(language.English, cases.NoLower)

	// Iterate over each file descriptor
	for _, proto := range protos {
		// Iterate over each service in the file
		for _, svc := range proto.GetService() {
			var s Service
			s.Name = svc.GetName()

			// Get the package alias if available
			alias, _ := getGoPackage(proto)
			if alias != "" {
				s.Package = alias + "."
			}

			// Populate the methods for the service
			methods := make([]methodTemplate, len(svc.Method))
			for j, method := range svc.Method {
				// Determine the type of method
				tipe := methodTypeStandard
				if method.GetServerStreaming() && !method.GetClientStreaming() {
					tipe = methodTypeServerStream
				} else if !method.GetServerStreaming() && method.GetClientStreaming() {
					tipe = methodTypeClientStream
				} else if method.GetServerStreaming() && method.GetClientStreaming() {
					tipe = methodTypeBidirectional
				}

				// Populate the methodTemplate struct
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
	// Split the message type into package and type parts
	split := strings.Split(tipe, ".")[1:]
	targetPackage := strings.Join(split[:len(split)-1], ".")
	targetType := split[len(split)-1]

	// Iterate over the protos to find the target message
	for _, proto := range protos {
		// Check if the proto package matches the target package
		if proto.GetPackage() != targetPackage {
			continue
		}

		// Iterate over the messages in the proto
		for _, msg := range proto.GetMessageType() {
			// Check if the message name matches the target type
			if msg.GetName() == targetType {
				// Get the package alias if available
				alias, _ := getGoPackage(proto)
				if alias != "" {
					alias += "."
				}

				// Return the fully qualified message type
				return fmt.Sprintf("%s%s", alias, msg.GetName())
			}
		}
	}

	// Return the target type if no match was found
	return targetType
}

// keywords is a map that contains all the reserved keywords in Go.
// It helps to determine if a given word is a keyword or not.
var keywords = map[string]bool{
	"break":       true,
	"case":        true,
	"chan":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"interface":   true,
	"map":         true,
	"package":     true,
	"range":       true,
	"return":      true,
	"select":      true,
	"struct":      true,
	"switch":      true,
	"type":        true,
	"var":         true,
	"bool":        true,
	"byte":        true,
	"complex128":  true,
	"complex64":   true,
	"error":       true,
	"float32":     true,
	"float64":     true,
	"int":         true,
	"int16":       true,
	"int32":       true,
	"int64":       true,
	"int8":        true,
	"rune":        true,
	"string":      true,
	"uint":        true,
	"uint16":      true,
	"uint32":      true,
	"uint64":      true,
	"uint8":       true,
	"uintptr":     true,
}

// isKeyword checks if a word is a keyword or not.
// It does a case insensitive comparison.
func isKeyword(word string) bool {
	_, ok := keywords[strings.ToLower(word)]

	return ok
}
