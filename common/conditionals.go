// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package common

import (
	"fmt"
	"infini.sh/framework/core/conditions"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/lib/fasthttp"
	log "github.com/cihub/seelog"
	"strings"
	"errors"
)

// NewConditional returns a constructor suitable for registering when conditionals as a plugin.
func NewConditional(
	ruleFactory RequestFilterConstructor,
) RequestFilterConstructor {
	return func(cfg *config.Config) (RequestFilter, error) {
		rule, err := ruleFactory(cfg)
		if err != nil {
			return nil, err
		}
		return addCondition(cfg, rule)
	}
}

// NewConditionList takes a slice of Config objects and turns them into real Condition objects.
func NewConditionList(config []conditions.Config) ([]conditions.Condition, error) {
	out := make([]conditions.Condition, len(config))
	for i, condConfig := range config {
		cond, err := conditions.NewCondition(&condConfig)
		if err != nil {
			return nil, err
		}

		out[i] = cond
	}
	return out, nil
}

// WhenProcessor is a tuple of condition plus a Processor.
type WhenProcessor struct {
	condition conditions.Condition
	p         RequestFilter
}

// NewConditionRule returns a processor that will execute the provided processor if the condition is true.
func NewConditionRule(
	config conditions.Config,
	p RequestFilter,
) (RequestFilter, error) {
	cond, err := conditions.NewCondition(&config)
	if err != nil {
		return nil, errors.Unwrap(err)
	}

	if cond == nil {
		return p, nil
	}
	return &WhenProcessor{cond, p}, nil
}

// Run executes this WhenProcessor.
func (r WhenProcessor) Process(ctx *fasthttp.RequestCtx) {
	if !ctx.ShouldContinue(){
		if global.Env().IsDebug{
			log.Debugf("filter [%v] not continued",r.Name())
		}
		ctx.AddFlowProcess(r.Name()+"-skipped")
		return
	}

	if !(r.condition).Check(ctx) {
		ctx.AddFlowProcess(r.p.Name()+"-skipped")
		return
	}
	ctx.AddFlowProcess(r.p.Name())
	r.p.Process(ctx)
}

func (r WhenProcessor) Name() string {
	return "when"
}

func (r *WhenProcessor) String() string {
	return fmt.Sprintf("%v, condition=%v", r.p.Name(), r.condition.String())
}

func addCondition(
	cfg *config.Config,
	p RequestFilter,
) (RequestFilter, error) {
	if !cfg.HasField("when") {
		return p, nil
	}
	sub, err := cfg.Child("when", -1)
	if err != nil {
		return nil, err
	}

	condConfig := conditions.Config{}
	if err := sub.Unpack(&condConfig); err != nil {
		return nil, err
	}

	return NewConditionRule(condConfig, p)
}

type ifThenElseConfig struct {
	Cond conditions.Config `config:"if"   validate:"required"`
	Then *config.Config    `config:"then" validate:"required"`
	Else *config.Config    `config:"else"`
}

// IfThenElseProcessor executes one set of processors (then) if the condition is
// true and another set of processors (else) if the condition is false.
type IfThenElseProcessor struct {
	cond conditions.Condition
	then *RequestFilters
	els  *RequestFilters
}

// NewIfElseThenProcessor construct a new IfThenElseProcessor.
func NewIfElseThenProcessor(cfg *config.Config) (*IfThenElseProcessor, error) {
	var tempConfig ifThenElseConfig
	if err := cfg.Unpack(&tempConfig); err != nil {
		return nil, err
	}

	cond, err := conditions.NewCondition(&tempConfig.Cond)
	if err != nil {
		return nil, err
	}

	newProcessors := func(c *config.Config) (*RequestFilters, error) {
		if c == nil {
			return nil, nil
		}
		if !c.IsArray() {
			return New([]*config.Config{c})
		}

		var pc PluginConfig
		if err := c.Unpack(&pc); err != nil {
			return nil, err
		}
		return New(pc)
	}

	var ifProcessors, elseProcessors *RequestFilters
	if ifProcessors, err = newProcessors(tempConfig.Then); err != nil {
		return nil, err
	}
	if elseProcessors, err = newProcessors(tempConfig.Else); err != nil {
		return nil, err
	}

	return &IfThenElseProcessor{cond, ifProcessors, elseProcessors}, nil
}

// Run checks the if condition and executes the processors attached to the
// then statement or the else statement based on the condition.
func (p IfThenElseProcessor) Process(ctx *fasthttp.RequestCtx) {
	if !ctx.ShouldContinue(){
		if global.Env().IsDebug{
			log.Debugf("filter [%v] not continued",p.Name())
		}
		ctx.AddFlowProcess("skipped")
		return
	}

	if p.cond.Check(ctx) {
		if global.Env().IsDebug{
			log.Trace("if -> then branch")
		}
		ctx.AddFlowProcess("then")
		p.then.Process( ctx)
	} else if p.els != nil {
		if global.Env().IsDebug {
			log.Trace("if -> else branch")
		}
		ctx.AddFlowProcess("else")
		p.els.Process( ctx)
	}
}

func (p IfThenElseProcessor) Name() string {
	return "if"
}

func (p *IfThenElseProcessor) String() string {
	var sb strings.Builder
	sb.WriteString("if ")
	sb.WriteString(p.cond.String())
	sb.WriteString(" then ")
	sb.WriteString(p.then.Name())
	if p.els != nil {
		sb.WriteString(" else ")
		sb.WriteString(p.els.Name())
	}
	return sb.String()
}
