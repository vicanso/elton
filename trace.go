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

// Start starts a sub trace and return done function
// for sub trace.
func (t *Trace) Start(name string) func() {
	startedAt := time.Now()
	info := &TraceInfo{
		Name: name,
	}
	t.Add(info)
	return func() {
		info.Duration = time.Since(startedAt)
	}
}

// Add adds trace info to trace
func (t *Trace) Add(info *TraceInfo) *Trace {
	t.Infos = append(t.Infos, info)
	return t
}

// Calculate calculates the duration of middleware.
// 中间件采用洋葱模型，前面中间件记录的时长包含了它调用Next()之后
// 所有后续中间件的时长。由于trace信息按执行顺序添加，
// 依次让每个中间件减去紧随其后的中间件时长，即得到各自的真实耗时。
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

// getMs formats ns as milliseconds string (for tests and callers needing string).
func getMs(ns int) string {
	var s strings.Builder
	writeMs(&s, ns)
	return s.String()
}

// writeMs appends duration in milliseconds (up to 2 fractional digits) without
// intermediate string allocations for the numeric part.
func writeMs(s *strings.Builder, ns int) {
	microSecond := int(time.Microsecond)
	milliSecond := int(time.Millisecond)
	if ns < microSecond {
		s.WriteByte('0')
		return
	}

	var buf [20]byte
	// 计算 ms 整数部分
	ms := ns / milliSecond
	s.Write(strconv.AppendInt(buf[:0], int64(ms), 10))

	// 计算 micro seconds（取小数点两位）
	offset := (ns % milliSecond) / microSecond
	unit := 10
	if offset < unit {
		return
	}
	s.WriteByte('.')
	if offset < 100 {
		s.WriteByte('0')
		s.Write(strconv.AppendInt(buf[:0], int64(offset/unit), 10))
		return
	}
	s.Write(strconv.AppendInt(buf[:0], int64(offset/unit), 10))
}

// ServerTiming return server timing with prefix
func (traceInfos TraceInfos) ServerTiming(prefix string) string {
	size := len(traceInfos)
	if size == 0 {
		return ""
	}

	// 转换为 http server timing
	s := new(strings.Builder)
	// 预估：prefix+index+固定片段+name+逗号
	s.Grow((len(prefix) + 24) * size)
	var idxBuf [20]byte
	for i, traceInfo := range traceInfos {
		v := traceInfo.Duration.Nanoseconds()
		s.WriteString(prefix)
		s.Write(strconv.AppendInt(idxBuf[:0], int64(i), 10))
		s.Write(ServerTimingDur)
		writeMs(s, int(v))
		s.Write(ServerTimingDesc)
		s.WriteString(traceInfo.Name)
		s.Write(ServerTimingEnd)
		if i != size-1 {
			s.WriteByte(',')
		}
	}
	return s.String()
}

// Filter filters the trace info, the new trace infos will be returned.
func (traceInfos TraceInfos) Filter(fn func(*TraceInfo) bool) TraceInfos {
	infos := make(TraceInfos, 0, len(traceInfos))
	for _, info := range traceInfos {
		if fn(info) {
			infos = append(infos, info)
		}
	}
	return infos
}

// FilterDurationGreaterThan filters the trace infos of which duration is greater than d.
func (traceInfos TraceInfos) FilterDurationGreaterThan(d time.Duration) TraceInfos {
	return traceInfos.Filter(func(ti *TraceInfo) bool {
		return ti.Duration > d
	})
}

// TraceFromContext gets trace from context, if context without trace, new trace will be created.
func TraceFromContext(ctx context.Context) *Trace {
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
