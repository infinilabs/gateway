// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package model

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonC80ae7adDecodeInfiniShGatewayCommonModel(in *jlexer.Lexer, out *Response) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "cached":
			out.Cached = bool(in.Bool())
		case "local_addr":
			out.LocalAddr = string(in.String())
		case "remote_addr":
			out.RemoteAddr = string(in.String())
		case "header":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				if !in.IsDelim('}') {
					out.Header = make(map[string]string)
				} else {
					out.Header = nil
				}
				for !in.IsDelim('}') {
					key := string(in.String())
					in.WantColon()
					var v1 string
					v1 = string(in.String())
					(out.Header)[key] = v1
					in.WantComma()
				}
				in.Delim('}')
			}
		case "status_code":
			out.StatusCode = int(in.Int())
		case "body_length":
			out.BodyLength = int(in.Int())
		case "body":
			out.Body = string(in.String())
		case "elapsed":
			out.ElapsedTimeInMs = float32(in.Float32())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonC80ae7adEncodeInfiniShGatewayCommonModel(out *jwriter.Writer, in Response) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"cached\":"
		out.RawString(prefix[1:])
		out.Bool(bool(in.Cached))
	}
	if in.LocalAddr != "" {
		const prefix string = ",\"local_addr\":"
		out.RawString(prefix)
		out.String(string(in.LocalAddr))
	}
	if in.RemoteAddr != "" {
		const prefix string = ",\"remote_addr\":"
		out.RawString(prefix)
		out.String(string(in.RemoteAddr))
	}
	if len(in.Header) != 0 {
		const prefix string = ",\"header\":"
		out.RawString(prefix)
		{
			out.RawByte('{')
			v2First := true
			for v2Name, v2Value := range in.Header {
				if v2First {
					v2First = false
				} else {
					out.RawByte(',')
				}
				out.String(string(v2Name))
				out.RawByte(':')
				out.String(string(v2Value))
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"status_code\":"
		out.RawString(prefix)
		out.Int(int(in.StatusCode))
	}
	{
		const prefix string = ",\"body_length\":"
		out.RawString(prefix)
		out.Int(int(in.BodyLength))
	}
	if in.Body != "" {
		const prefix string = ",\"body\":"
		out.RawString(prefix)
		out.String(string(in.Body))
	}
	{
		const prefix string = ",\"elapsed\":"
		out.RawString(prefix)
		out.Float32(float32(in.ElapsedTimeInMs))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v Response) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v Response) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *Response) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *Response) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel(l, v)
}
func easyjsonC80ae7adDecodeInfiniShGatewayCommonModel1(in *jlexer.Lexer, out *Request) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "started":
			out.StartTime = string(in.String())
		case "host":
			out.Host = string(in.String())
		case "remote_addr":
			out.RemoteAddr = string(in.String())
		case "local_addr":
			out.LocalAddr = string(in.String())
		case "method":
			out.Method = string(in.String())
		case "header":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				if !in.IsDelim('}') {
					out.Header = make(map[string]string)
				} else {
					out.Header = nil
				}
				for !in.IsDelim('}') {
					key := string(in.String())
					in.WantColon()
					var v3 string
					v3 = string(in.String())
					(out.Header)[key] = v3
					in.WantComma()
				}
				in.Delim('}')
			}
		case "uri":
			out.URI = string(in.String())
		case "path":
			out.Path = string(in.String())
		case "query_args":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				if !in.IsDelim('}') {
					out.QueryArgs = make(map[string]string)
				} else {
					out.QueryArgs = nil
				}
				for !in.IsDelim('}') {
					key := string(in.String())
					in.WantColon()
					var v4 string
					v4 = string(in.String())
					(out.QueryArgs)[key] = v4
					in.WantComma()
				}
				in.Delim('}')
			}
		case "body_length":
			out.BodyLength = int(in.Int())
		case "body":
			out.Body = string(in.String())
		case "user":
			out.User = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonC80ae7adEncodeInfiniShGatewayCommonModel1(out *jwriter.Writer, in Request) {
	out.RawByte('{')
	first := true
	_ = first
	if in.StartTime != "" {
		const prefix string = ",\"started\":"
		first = false
		out.RawString(prefix[1:])
		out.String(string(in.StartTime))
	}
	if in.Host != "" {
		const prefix string = ",\"host\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Host))
	}
	if in.RemoteAddr != "" {
		const prefix string = ",\"remote_addr\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.RemoteAddr))
	}
	if in.LocalAddr != "" {
		const prefix string = ",\"local_addr\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.LocalAddr))
	}
	if in.Method != "" {
		const prefix string = ",\"method\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Method))
	}
	if len(in.Header) != 0 {
		const prefix string = ",\"header\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('{')
			v5First := true
			for v5Name, v5Value := range in.Header {
				if v5First {
					v5First = false
				} else {
					out.RawByte(',')
				}
				out.String(string(v5Name))
				out.RawByte(':')
				out.String(string(v5Value))
			}
			out.RawByte('}')
		}
	}
	if in.URI != "" {
		const prefix string = ",\"uri\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.URI))
	}
	if in.Path != "" {
		const prefix string = ",\"path\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Path))
	}
	if len(in.QueryArgs) != 0 {
		const prefix string = ",\"query_args\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('{')
			v6First := true
			for v6Name, v6Value := range in.QueryArgs {
				if v6First {
					v6First = false
				} else {
					out.RawByte(',')
				}
				out.String(string(v6Name))
				out.RawByte(':')
				out.String(string(v6Value))
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"body_length\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.BodyLength))
	}
	if in.Body != "" {
		const prefix string = ",\"body\":"
		out.RawString(prefix)
		out.String(string(in.Body))
	}
	if in.User != "" {
		const prefix string = ",\"user\":"
		out.RawString(prefix)
		out.String(string(in.User))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v Request) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v Request) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *Request) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *Request) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel1(l, v)
}
func easyjsonC80ae7adDecodeInfiniShGatewayCommonModel2(in *jlexer.Lexer, out *HttpRequest) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = uint64(in.Uint64())
		case "timestamp":
			out.LoggingTime = string(in.String())
		case "local_ip":
			out.LocalIP = string(in.String())
		case "remote_ip":
			out.RemoteIP = string(in.String())
		case "tls":
			out.IsTLS = bool(in.Bool())
		case "tls_reuse":
			out.TLSDidResume = bool(in.Bool())
		case "request":
			if in.IsNull() {
				in.Skip()
				out.Request = nil
			} else {
				if out.Request == nil {
					out.Request = new(Request)
				}
				(*out.Request).UnmarshalEasyJSON(in)
			}
		case "response":
			if in.IsNull() {
				in.Skip()
				out.Response = nil
			} else {
				if out.Response == nil {
					out.Response = new(Response)
				}
				(*out.Response).UnmarshalEasyJSON(in)
			}
		case "flow":
			if in.IsNull() {
				in.Skip()
				out.DataFlow = nil
			} else {
				if out.DataFlow == nil {
					out.DataFlow = new(DataFlow)
				}
				(*out.DataFlow).UnmarshalEasyJSON(in)
			}
		case "elastic":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				if !in.IsDelim('}') {
					out.Elastic = make(map[string]interface{})
				} else {
					out.Elastic = nil
				}
				for !in.IsDelim('}') {
					key := string(in.String())
					in.WantColon()
					var v7 interface{}
					if m, ok := v7.(easyjson.Unmarshaler); ok {
						m.UnmarshalEasyJSON(in)
					} else if m, ok := v7.(json.Unmarshaler); ok {
						_ = m.UnmarshalJSON(in.Raw())
					} else {
						v7 = in.Interface()
					}
					(out.Elastic)[key] = v7
					in.WantComma()
				}
				in.Delim('}')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonC80ae7adEncodeInfiniShGatewayCommonModel2(out *jwriter.Writer, in HttpRequest) {
	out.RawByte('{')
	first := true
	_ = first
	if in.ID != 0 {
		const prefix string = ",\"id\":"
		first = false
		out.RawString(prefix[1:])
		out.Uint64(uint64(in.ID))
	}
	if in.LoggingTime != "" {
		const prefix string = ",\"timestamp\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.LoggingTime))
	}
	if in.LocalIP != "" {
		const prefix string = ",\"local_ip\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.LocalIP))
	}
	if in.RemoteIP != "" {
		const prefix string = ",\"remote_ip\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.RemoteIP))
	}
	{
		const prefix string = ",\"tls\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.IsTLS))
	}
	if in.TLSDidResume {
		const prefix string = ",\"tls_reuse\":"
		out.RawString(prefix)
		out.Bool(bool(in.TLSDidResume))
	}
	if in.Request != nil {
		const prefix string = ",\"request\":"
		out.RawString(prefix)
		(*in.Request).MarshalEasyJSON(out)
	}
	if in.Response != nil {
		const prefix string = ",\"response\":"
		out.RawString(prefix)
		(*in.Response).MarshalEasyJSON(out)
	}
	if in.DataFlow != nil {
		const prefix string = ",\"flow\":"
		out.RawString(prefix)
		(*in.DataFlow).MarshalEasyJSON(out)
	}
	if len(in.Elastic) != 0 {
		const prefix string = ",\"elastic\":"
		out.RawString(prefix)
		{
			out.RawByte('{')
			v8First := true
			for v8Name, v8Value := range in.Elastic {
				if v8First {
					v8First = false
				} else {
					out.RawByte(',')
				}
				out.String(string(v8Name))
				out.RawByte(':')
				if m, ok := v8Value.(easyjson.Marshaler); ok {
					m.MarshalEasyJSON(out)
				} else if m, ok := v8Value.(json.Marshaler); ok {
					out.Raw(m.MarshalJSON())
				} else {
					out.Raw(json.Marshal(v8Value))
				}
			}
			out.RawByte('}')
		}
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v HttpRequest) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel2(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v HttpRequest) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel2(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *HttpRequest) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel2(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *HttpRequest) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel2(l, v)
}
func easyjsonC80ae7adDecodeInfiniShGatewayCommonModel3(in *jlexer.Lexer, out *DataFlow) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "from":
			out.From = string(in.String())
		case "relay":
			out.Relay = string(in.String())
		case "to":
			if in.IsNull() {
				in.Skip()
				out.To = nil
			} else {
				in.Delim('[')
				if out.To == nil {
					if !in.IsDelim(']') {
						out.To = make([]string, 0, 4)
					} else {
						out.To = []string{}
					}
				} else {
					out.To = (out.To)[:0]
				}
				for !in.IsDelim(']') {
					var v9 string
					v9 = string(in.String())
					out.To = append(out.To, v9)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "process":
			if in.IsNull() {
				in.Skip()
				out.Process = nil
			} else {
				in.Delim('[')
				if out.Process == nil {
					if !in.IsDelim(']') {
						out.Process = make([]string, 0, 4)
					} else {
						out.Process = []string{}
					}
				} else {
					out.Process = (out.Process)[:0]
				}
				for !in.IsDelim(']') {
					var v10 string
					v10 = string(in.String())
					out.Process = append(out.Process, v10)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonC80ae7adEncodeInfiniShGatewayCommonModel3(out *jwriter.Writer, in DataFlow) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"from\":"
		out.RawString(prefix[1:])
		out.String(string(in.From))
	}
	{
		const prefix string = ",\"relay\":"
		out.RawString(prefix)
		out.String(string(in.Relay))
	}
	{
		const prefix string = ",\"to\":"
		out.RawString(prefix)
		if in.To == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v11, v12 := range in.To {
				if v11 > 0 {
					out.RawByte(',')
				}
				out.String(string(v12))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"process\":"
		out.RawString(prefix)
		if in.Process == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v13, v14 := range in.Process {
				if v13 > 0 {
					out.RawByte(',')
				}
				out.String(string(v14))
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v DataFlow) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel3(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v DataFlow) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonC80ae7adEncodeInfiniShGatewayCommonModel3(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *DataFlow) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel3(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *DataFlow) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonC80ae7adDecodeInfiniShGatewayCommonModel3(l, v)
}