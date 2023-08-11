package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Context for use with multiple config files
type Context struct {
	Current   string `yaml:"current"`
	Directory string `yaml:"directory"`
}

// ContextInit generates and returns default context
func ContextInit() (*Context, error) {
	context := &Context{Current: defaults.DefaultContext, Directory: defaults.DefaultContextDirectory}
	return context, nil
}

// GetCurrentConfig return path to config file
func (ctx *Context) GetCurrentConfig() string {
	edenDir, err := DefaultEdenDir()
	if err != nil {
		log.Fatalf("GetCurrentConfig DefaultEdenDir error: %s", err)
	}
	return filepath.Join(edenDir, ctx.Directory, fmt.Sprintf("%s.yml", ctx.Current))
}

// SetContext set current contexts
func (ctx *Context) SetContext(context string) {
	ctx.Current = context
	ctx.Save()
}

// ListContexts show available contexts
func (ctx *Context) ListContexts() (contexts []string) {
	edenDir, err := DefaultEdenDir()
	if err != nil {
		log.Fatalf("GetCurrentConfig DefaultEdenDir error: %s", err)
	}
	files, err := os.ReadDir(filepath.Join(edenDir, ctx.Directory))
	if err != nil {
		log.Fatalf("ListContexts ReadDir error: %s", err)
	}

	for _, file := range files {
		contexts = append(contexts, strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name())))
	}
	return
}

// Save save file with context data
func (ctx *Context) Save() {
	edenDir, err := DefaultEdenDir()
	if err != nil {
		log.Fatalf("Context Save DefaultEdenDir error: %s", err)
	}
	contextDirectory := filepath.Join(edenDir, ctx.Directory)
	if err := os.MkdirAll(contextDirectory, 0755); err != nil {
		log.Fatalf("MkdirAll(%s) error: %s", contextDirectory, err)
	}
	data, err := yaml.Marshal(ctx)
	if err != nil {
		log.Fatalf("Context Marshal error: %s", err)
	}
	contextFile := filepath.Join(edenDir, defaults.DefaultContextFile)
	if err := os.WriteFile(contextFile, data, 0755); err != nil {
		log.Fatalf("Write Context File %s error: %s", contextFile, err)
	}
}

// ContextLoad read file with context data
func ContextLoad() (*Context, error) {
	edenDir, err := DefaultEdenDir()
	if err != nil {
		return nil, fmt.Errorf("context Load DefaultEdenDir error: %s", err)
	}
	edenConfigEnv := os.Getenv(defaults.DefaultConfigEnv)
	if edenConfigEnv != "" {
		ctx, err := ContextInit()
		if err != nil {
			return nil, fmt.Errorf("ContextInit error: %s", err)
		}
		ctx.Current = edenConfigEnv
		return ctx, nil
	}
	contextFile := filepath.Join(edenDir, defaults.DefaultContextFile)
	if _, err := os.Stat(contextFile); os.IsNotExist(err) {
		return ContextInit()
	}
	buf, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, fmt.Errorf("read context file %s error: %s", contextFile, err)
	}
	ctx, err := ContextInit()
	if err != nil {
		return nil, fmt.Errorf("ContextInit error: %s", err)
	}
	if err := yaml.Unmarshal(buf, ctx); err != nil {
		return nil, fmt.Errorf("read Context File %s error: %s", contextFile, err)
	}
	return ctx, nil
}
