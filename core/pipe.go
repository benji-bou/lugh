package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"text/template"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
	"gopkg.in/yaml.v2"
)

type Pipeable interface {
	Pipe(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error)
}

type ChainedPipe struct {
	chain []Pipeable
}

func NewChainedPipe(chain ...Pipeable) Pipeable {
	return ChainedPipe{chain}
}

func (cp ChainedPipe) Pipe(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	errorsC := make([]<-chan error, 0, len(cp.chain))
	for _, elem := range cp.chain {
		var errC <-chan error
		input, errC = elem.Pipe(ctx, input)
		if errC != nil {
			errorsC = append(errorsC, errC)
		}
	}
	return input, chantools.Merge(errorsC...)
}

type PipeFunc func(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error)

func (pf PipeFunc) Pipe(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return pf(ctx, input)
}

func NewMapPipe(mapper func(elem *pluginctl.DataStream) *pluginctl.DataStream) Pipeable {
	return NewWorkerPipe(func(elem *pluginctl.DataStream, c chan<- *pluginctl.DataStream) {
		c <- mapper(elem)
	})

}

func NewWorkerPipe(worker func(elem *pluginctl.DataStream, c chan<- *pluginctl.DataStream)) Pipeable {
	return PipeFunc(func(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
		return chantools.NewWithErr(func(c chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {
			for {
				select {
				case i, ok := <-input:
					if !ok {
						return
					}
					worker(i, c)
				case <-ctx.Done():
					return
				}
			}
		})
	})
}

func NewEmptyPipe() Pipeable {
	return PipeFunc(func(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
		return input, make(<-chan error)
	})
}

type PluginPipe struct {
	to pluginctl.SecPipelinePluginable
}

func NewPluginPipe(to pluginctl.SecPipelinePluginable) Pipeable {
	return PluginPipe{to: to}
}

func (dc PluginPipe) Pipe(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return dc.to.Run(ctx, input)
}

type GoTemplateOption = helper.OptionError[goTemplatePipe]

type goTemplatePipe struct {
	template    *template.Template
	unmarshaler func(in []byte, v any) error
}

func WithJsonInput() GoTemplateOption {
	return func(configure *goTemplatePipe) error {
		configure.unmarshaler = json.Unmarshal
		return nil
	}
}

func WithYamlInput() GoTemplateOption {
	return func(configure *goTemplatePipe) error {
		configure.unmarshaler = yaml.Unmarshal
		return nil
	}
}

func WithTemplate(tmp *template.Template) GoTemplateOption {
	return func(configure *goTemplatePipe) error {
		configure.template = tmp
		return nil
	}
}

func WithTemplatePattern(pattern string) GoTemplateOption {
	return func(configure *goTemplatePipe) error {
		tmp, err := template.New("GoTemplatePipePattern").Parse(pattern)
		if err != nil {
			return fmt.Errorf("couldn't parse go template pattern: %w", err)
		}
		configure.template = tmp
		return nil
	}
}

type GoTemplateConfig struct {
	Format  string `yaml:"format"`
	Pattern string `yaml:"pattern"`
}

func NewGoTemplatePipeWithConfig(config GoTemplateConfig) Pipeable {
	opt := []GoTemplateOption{}
	switch config.Format {
	case "yaml":
		opt = append(opt, WithYamlInput())
	default:
		opt = append(opt, WithJsonInput())
	}
	opt = append(opt, WithTemplatePattern(config.Pattern))
	return NewGoTemplatePipe(opt...)
}

func NewGoTemplatePipe[OPT GoTemplateOption](opt ...OPT) Pipeable {

	tmp, err := helper.ConfigureWithError(goTemplatePipe{}, opt...)
	if err != nil {
		slog.Error("failed to generate go template pipe, fallback to default pipe", "error", err)

		return NewEmptyPipe()
	}
	if tmp.template == nil {
		return NewEmptyPipe()
	}
	return NewMapPipe(func(elem *pluginctl.DataStream) *pluginctl.DataStream {
		var dataInput map[string]any
		err := tmp.unmarshaler(elem.Data, &dataInput)
		if err != nil {
			slog.Error("failed to unmarshal pipe elem", "error", err, "pipe", "GoTemplatePipe")
			return elem
		}
		buff := &bytes.Buffer{}
		err = tmp.template.ExecuteTemplate(buff, tmp.template.Name(), dataInput)
		if err != nil {
			slog.Error("failed to execute template", "error", err, "pipe", "GoTemplatePipe", "inputdata", dataInput)
			return elem
		}
		return &pluginctl.DataStream{
			Data:       buff.Bytes(),
			ParentSrc:  elem.ParentSrc + "_templated",
			Id:         elem.Id,
			IsComplete: elem.IsComplete,
			TotalLen:   int64(buff.Len()),
		}
	})
}

func NewBase64Decoder() Pipeable {
	return NewMapPipe(func(elem *pluginctl.DataStream) *pluginctl.DataStream {
		res, err := base64.RawStdEncoding.DecodeString(string(elem.Data))
		if err != nil {
			slog.Error("unable to decode base64", "error", err)
			return elem
		}
		return &pluginctl.DataStream{
			Data:       res,
			ParentSrc:  elem.ParentSrc + "_base64decoded",
			Id:         elem.Id,
			IsComplete: elem.IsComplete,
			TotalLen:   int64(len(res)),
		}
	})
}

func NewRegexpPipe(regexpPattern string, n int) Pipeable {
	regex, err := regexp.Compile(regexpPattern)
	if err != nil {
		slog.Warn("regex pipe failed to compile regexp pattern", "error", err)
		return NewEmptyPipe()
	}
	return NewMapPipe(func(elem *pluginctl.DataStream) *pluginctl.DataStream {

		resAllArr := regex.FindAllSubmatch(elem.Data, -1)
		resArr := make([][]byte, 0, len(resAllArr))
		for _, match := range resAllArr {
			if len(match) == 0 {
				continue
			} else if n >= len(match) {
				resArr = append(resArr, match[0])
			} else {
				resArr = append(resArr, match[n])
			}
		}
		res := bytes.Join(resArr, []byte("\n"))
		return &pluginctl.DataStream{
			Data:       res,
			ParentSrc:  elem.ParentSrc + "_regex",
			Id:         elem.Id,
			IsComplete: elem.IsComplete,
			TotalLen:   int64(len(res)),
		}
	})
}

func NewInsertStringPipe(insert string) Pipeable {

	return NewMapPipe(func(elem *pluginctl.DataStream) *pluginctl.DataStream {

		res := append(elem.Data, []byte(insert)...)
		return &pluginctl.DataStream{
			Data:       res,
			ParentSrc:  elem.ParentSrc + "_regex",
			Id:         elem.Id,
			IsComplete: elem.IsComplete,
			TotalLen:   int64(len(res)),
		}
	})
}

func NewSplitPipe(sep string) Pipeable {
	return NewWorkerPipe(func(elem *pluginctl.DataStream, c chan<- *pluginctl.DataStream) {
		inputStr := string(elem.Data)
		res := strings.Split(inputStr, sep)
		for _, r := range res {
			resDataStream := &pluginctl.DataStream{
				Data:       []byte(r),
				ParentSrc:  elem.ParentSrc + "_split",
				Id:         elem.Id,
				IsComplete: elem.IsComplete,
				TotalLen:   int64(len(r)),
			}
			c <- resDataStream
		}
	})
}
