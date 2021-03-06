package tmpl

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"

	gateway "github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/pkg/errors"
)

type generator struct {
	config  Config
	request *plugin.CodeGeneratorRequest
}

// New returns a new generator for the given template.
func Generate(request *plugin.CodeGeneratorRequest, config Config) (*plugin.CodeGeneratorResponse, error) {
	if len(request.FileToGenerate) == 0 {
		return nil, errors.New("no input files")
	}

	registry := gateway.NewRegistry()
	err := registry.Load(request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load request")
	}

	g := &generator{request: request, config: config}
	return g.Generate(), nil
}

func (g *generator) Generate() *plugin.CodeGeneratorResponse {
	if len(g.config.Operations) == 0 {
		g.config.Operations = defaultOperations(g.request)
	}

	response := &plugin.CodeGeneratorResponse{}
	errs := new(bytes.Buffer)
	for _, opConfig := range g.config.Operations {
		f, err := g.genTarget(opConfig)
		if err != nil {
			errs.WriteString(fmt.Sprintf("%s\n", err))
			continue
		}
		response.File = append(response.File, f)
	}

	if errs.Len() > 0 {
		response.File = nil
		response.Error = proto.String(errs.String())
	}
	return response
}

func defaultOperations(request *plugin.CodeGeneratorRequest) []OperationConfig {
	ops := []OperationConfig{
		{
			Template: "index.fragment.html",
			Output:   "index.fragment.html",
		},
	}
	for _, protoFile := range request.ProtoFile {
		op := OperationConfig{
			Template: "template.html",
			Target:   *protoFile.Name,
			Output:   fmt.Sprintf("%s.html", trimExt(*protoFile.Name)),
		}
		ops = append(ops, op)
	}
	return ops
}

type templateContext struct {
	*plugin.CodeGeneratorRequest
	Target *descriptor.FileDescriptorProto
}

func (g *generator) genTarget(opConfig OperationConfig) (*plugin.CodeGeneratorResponse_File, error) {
	protoFile := getProtoFileFromTarget(opConfig.Target, g.request)
	if opConfig.Target != "" && protoFile == nil {
		return nil, errors.Errorf("no input proto file for generator target %q", opConfig.Target)
	}

	tmpl, err := g.loadTemplate(opConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load template %s", opConfig.Template)
	}

	buf := new(bytes.Buffer)
	funcs := &tmplFuncs{
		protoFileDescriptor: protoFile,
		outputFile:          opConfig.Output,
		urlRoot:             g.config.URLRoot,
		protoFiles:          g.request.GetProtoFile(),
	}
	ctx := templateContext{
		CodeGeneratorRequest: g.request,
		Target:               protoFile,
	}
	err = tmpl.Funcs(funcs.funcMap()).Execute(buf, ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to render template")
	}

	return &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(opConfig.Output),
		Content: proto.String(buf.String()),
	}, nil
}

func getProtoFileFromTarget(target string, request *plugin.CodeGeneratorRequest) *descriptor.FileDescriptorProto {
	for _, v := range request.GetProtoFile() {
		if target == v.GetName() {
			return v
		}
	}
	return nil
}

func (g *generator) loadTemplate(opConfig OperationConfig) (*template.Template, error) {
	fullPath := filepath.Join(g.config.TemplateRoot, opConfig.Template)
	tmpl, err := template.New("main").Funcs(newDefaultTemplateFuncs()).ParseFiles(fullPath)
	if err != nil {
		return nil, err
	}
	return tmpl.Lookup(filepath.Base(fullPath)), nil
}
