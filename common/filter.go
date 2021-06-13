package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"strings"
)

type RequestFilterConstructor func(config *config.Config) (RequestFilter, error)


type RequestFilter interface {
	Name() string
	Process(ctx *fasthttp.RequestCtx)
}


type RequestFilters struct {
	List []RequestFilter
}


type Closer interface {
	Close() error
}

func Close(p RequestFilter) error {
	if closer, ok := p.(Closer); ok {
		return closer.Close()
	}
	return nil
}

func NewList() *RequestFilters {
	return &RequestFilters{}
}

func New(config PluginConfig) (*RequestFilters, error) {
	procs := NewList()

	for _, procConfig := range config {
		// Handle if/then/else processor which has multiple top-level keys.
		if procConfig.HasField("if") {
			p, err := NewIfElseThenProcessor(procConfig)
			if err != nil {
				return nil, errors.Wrap(err, "failed to make if/then/else processor")
			}
			procs.AddProcessor(p)
			continue
		}

		if len(procConfig.GetFields()) != 1 {
			return nil, errors.Errorf("each processor must have exactly one "+
				"action, but found %d actions (%v)",
				len(procConfig.GetFields()),
				strings.Join(procConfig.GetFields(), ","))
		}

		actionName := procConfig.GetFields()[0]
		actionCfg, err := procConfig.Child(actionName, -1)
		if err != nil {
			return nil, err
		}

		log.Debug("action:",actionName,",",actionCfg)

		if !actionCfg.HasField("_meta:config_id"){
			actionCfg.SetString("_meta:config_id",-1,util.GetUUID())
		}

		f:= GetFilterInstanceWithConfigV2(actionName,actionCfg)

		procs.AddProcessor(f)

		//gen, exists := registry.reg[actionName]
		//if !exists {
		//	var validActions []string
		//	for k := range registry.reg {
		//		validActions = append(validActions, k)
		//
		//	}
		//	return nil, errors.Errorf("the processor action %s does not exist. Valid actions: %v", actionName, strings.Join(validActions, ", "))
		//}
		//
		////actionCfg.PrintDebugf("Configure processor action '%v' with:", actionName)
		//constructor := gen.Plugin()
		//plugin, err := constructor(actionCfg)
		//if err != nil {
		//	return nil, err
		//}
		//
		//procs.AddProcessor(plugin)
	}

	if len(procs.List) > 0 {
		log.Debugf("Generated new processors: %v", procs)
	}
	return procs, nil
}

func (procs *RequestFilters) AddProcessor(p RequestFilter) {
	procs.List = append(procs.List, p)
}

func (procs *RequestFilters) AddRequestFilters(p RequestFilters) {
	// Subtlety: it is important here that we append the individual elements of
	// p, rather than p itself, even though
	// p implements the processors.Processor interface. This is
	// because the contents of what we return are later pulled out into a
	// processing.group rather than a processors.RequestFilters, and the two have
	// different error semantics: processors.RequestFilters aborts processing on
	// any error, whereas processing.group only aborts on fatal errors. The
	// latter is the most common behavior, and the one we are preserving here for
	// backwards compatibility.
	// We are unhappy about this and have plans to fix this inconsistency at a
	// higher level, but for now we need to respect the existing semantics.
	procs.List = append(procs.List, p.List...)
}

func (procs *RequestFilters) All() []RequestFilter {
	if procs == nil || len(procs.List) == 0 {
		return nil
	}

	ret := make([]RequestFilter, len(procs.List))
	for i, p := range procs.List {
		ret[i] = p
	}
	return ret
}

func (procs *RequestFilters) Close() error {
	return nil
	//var errs multierror.Errors
	//for _, p := range procs.List {
	//	err := Close(p)
	//	if err != nil {
	//		errs = append(errs, err)
	//	}
	//}
	//return errs.Err()
}

// Run executes the all processors serially and returns the event and possibly
// an error. If the event has been dropped (canceled) by a processor in the
// list then a nil event is returned.
func (procs *RequestFilters) Process(ctx *fasthttp.RequestCtx) {
	//var err error

	for _, p := range procs.List {

		if !ctx.ShouldContinue(){
			if global.Env().IsDebug{
				log.Debugf("filter [%v] not continued",p.Name())
			}
			ctx.AddFlowProcess("skipped")
			return
		}
		ctx.AddFlowProcess(p.Name())
		p.Process(ctx)
		//event, err = p.Process(filterCfg,ctx)
		//if err != nil {
		//	return event, errors.Wrapf(err, "failed applying processor %v", p)
		//}
		//if event == nil {
		//	// Drop.
		//	return nil, nil
		//}
	}
	//return event, nil
}

func (procs *RequestFilters) Name() string {
	return "filters"
}
func (procs *RequestFilters) String() string {
	var s []string
	for _, p := range procs.List {
		s = append(s, p.Name())
	}
	return strings.Join(s, ", ")
}
