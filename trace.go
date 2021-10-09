// MIT License

// Copyright (c) 2021 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package elton

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type (
	// TraceInfo trace's info
	TraceInfo struct {
		Middleware bool          `json:"-"`
		Name       string        `json:"name,omitempty"`
		Duration   time.Duration `json:"duration,omitempty"`
	}
	// TraceInfos trace infos
	TraceInfos []*TraceInfo

	Trace struct {
		calculateDone bool
		Infos         TraceInfos
	}
)

// NewTrace returns a new trace
func NewTrace() *Trace {
	return &Trace{
		Infos: make(TraceInfos, 0),
	}
}

// Add adds trace info to trace
func (t *Trace) Add(info *TraceInfo) *Trace {
	t.Infos = append(t.Infos, info)
	return t
}

// Calculate calculates the duration of middleware
func (t *Trace) Calculate() {
	if t.calculateDone {
		return
	}
	// middleware需要减去后面middleware的处理时长
	var cur *TraceInfo
	for _, item := range t.Infos {
		if !item.Middleware {
			continue
		}
		if cur != nil {
			cur.Duration -= item.Duration
		}
		cur = item
	}
	t.calculateDone = true
}

func getMs(ns int) string {
	microSecond := int(time.Microsecond)
	milliSecond := int(time.Millisecond)
	if ns < microSecond {
		return "0"
	}

	// 计算ms的位
	ms := ns / milliSecond
	prefix := strconv.Itoa(ms)

	// 计算micro seconds
	offset := (ns % milliSecond) / microSecond
	// 如果小于10，不展示小数点（取小数点两位）
	unit := 10
	if offset < unit {
		return prefix
	}
	// 如果小于100，补一位0
	if offset < 100 {
		return prefix + ".0" + strconv.Itoa(offset/unit)
	}
	return prefix + "." + strconv.Itoa(offset/unit)
}

// ServerTiming return server timing with prefix
func (traceInfos TraceInfos) ServerTiming(prefix string) string {
	size := len(traceInfos)
	if size == 0 {
		return ""
	}

	// 转换为 http server timing
	s := new(strings.Builder)
	// 每一个server timing长度预估为30
	s.Grow(30 * size)
	for i, traceInfo := range traceInfos {
		v := traceInfo.Duration.Nanoseconds()
		s.WriteString(prefix)
		s.WriteString(strconv.Itoa(i))
		s.Write(ServerTimingDur)
		s.WriteString(getMs(int(v)))
		s.Write(ServerTimingDesc)
		s.WriteString(traceInfo.Name)
		s.Write(ServerTimingEnd)
		if i != size-1 {
			s.WriteRune(',')
		}

	}
	return s.String()
}

// GetTrace get trace from context, if context without trace, new trace will be created.
func GetTrace(ctx context.Context) *Trace {
	value := ctx.Value(ContextTraceKey)
	if value == nil {
		return NewTrace()
	}
	trace, ok := value.(*Trace)
	if !ok {
		return NewTrace()
	}
	return trace
}
