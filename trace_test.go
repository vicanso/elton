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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrace(t *testing.T) {
	assert := assert.New(t)

	trace := NewTrace()

	trace.Add(&TraceInfo{
		Name:       "1",
		Middleware: true,
		Duration:   5 * time.Millisecond,
	})

	trace.Add(&TraceInfo{
		Name:       "2",
		Middleware: true,
		Duration:   3 * time.Millisecond,
	})

	trace.Add(&TraceInfo{
		Name:       "3",
		Middleware: true,
		Duration:   2 * time.Millisecond,
	})
	trace.Add(&TraceInfo{
		Name:     "3",
		Duration: 1 * time.Millisecond,
	})
	trace.Calculate()

	assert.Equal(2*time.Millisecond, trace.Infos[0].Duration)
	assert.Equal(time.Millisecond, trace.Infos[1].Duration)
	assert.Equal(2*time.Millisecond, trace.Infos[2].Duration)
	assert.Equal(time.Millisecond, trace.Infos[3].Duration)
}

func TestTraceStart(t *testing.T) {
	assert := assert.New(t)

	trace := NewTrace()
	done := trace.Start("test")
	time.Sleep(2 * time.Millisecond)
	done()
	assert.Equal(1, len(trace.Infos))
	assert.Equal("test", trace.Infos[0].Name)
	assert.True(trace.Infos[0].Duration > 1)
}

func TestConvertToServerTiming(t *testing.T) {
	assert := assert.New(t)
	traceInfos := make(TraceInfos, 0)

	t.Run("get ms", func(t *testing.T) {
		assert.Equal("0", getMs(10))
		assert.Equal("0.10", getMs(100000))
	})

	t.Run("empty trace infos", func(t *testing.T) {
		assert.Empty(traceInfos.ServerTiming(""), "no trace should return nil")
	})
	t.Run("server timing", func(t *testing.T) {
		traceInfos = append(traceInfos, &TraceInfo{
			Name:     "a",
			Duration: time.Microsecond * 10,
		})
		traceInfos = append(traceInfos, &TraceInfo{
			Name:     "b",
			Duration: time.Millisecond + time.Microsecond,
		})
		assert.Equal(`elton-0;dur=0.01;desc="a",elton-1;dur=1;desc="b"`, string(traceInfos.ServerTiming("elton-")))
	})
}

func TestGetTrace(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	// 第一次无则创建
	t1 := GetTrace(ctx)
	assert.NotNil(t1)
	t1.Add(&TraceInfo{})

	// 第二次创建时与第一次非同一个值
	t2 := GetTrace(ctx)
	assert.NotNil(t2)
	assert.NotEqual(t1, t2)

	ctx = context.WithValue(ctx, ContextTraceKey, "a")
	t3 := GetTrace(ctx)
	assert.NotNil(t3)

	ctx = context.WithValue(ctx, ContextTraceKey, t2)
	t4 := GetTrace(ctx)
	assert.Equal(t2, t4)
}

func TestTraceFilter(t *testing.T) {
	assert := assert.New(t)

	traceInfos := TraceInfos{
		{
			Name:     "a",
			Duration: time.Millisecond,
		},
		{
			Name:     "b",
			Duration: 10 * time.Millisecond,
		},
	}
	assert.Equal(TraceInfos{
		{
			Name:     "a",
			Duration: time.Millisecond,
		},
	}, traceInfos.Filter(func(ti *TraceInfo) bool {
		return ti.Name == "a"
	}))

	assert.Equal(TraceInfos{
		{
			Name:     "b",
			Duration: 10 * time.Millisecond,
		},
	}, traceInfos.FilterDurationGT(5*time.Millisecond))
}
