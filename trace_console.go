package goservice

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/fatih/color"
)

func initTraceConsole() *traceConsole {
	s := &traceConsole{
		Width:      120,
		GaugeWidth: 40,
	}
	return s
}

type PrintSpanParent struct {
	Name         string
	Traceid      string
	Callinglevel int
	Last         bool
}

type traceConsole struct {
	Width      int
	GaugeWidth int
}

func (s *traceConsole) ExportSpan(spans []*traceSpan) {
	if len(spans) == 0 {
		return
	}

	startTime := spans[0].StartTime
	finishtime := spans[0].FinishTime
	deep := spans[0].Tags.CallingLevel
	for _, v := range spans {
		if startTime >= v.StartTime {
			startTime = v.StartTime
		}
		if finishtime <= v.FinishTime {
			finishtime = v.FinishTime
		}
		if deep <= v.Tags.CallingLevel {
			deep = v.Tags.CallingLevel
		}
	}

	duration := finishtime - startTime
	spans = s.sortSpans(spans)

	s.drawTableTop()

	titleId := "ID: " + spans[0].Tags.RequestId
	titleEnd := "total: " + strconv.Itoa(len(spans)) + ", deep: " + strconv.Itoa(deep)
	s.drawLine(s.getAlignedTexts(titleId+r(" ", s.Width-len(titleId)-len(titleEnd)-4)+titleEnd, s.Width-4), false)
	s.drawHorizonalLine()
	s.drawSpan(spans, spans[0], 0, float64(startTime), float64(duration))
	s.drawTableBottom()
}

func (s *traceConsole) sortSpans(spans []*traceSpan) []*traceSpan {
	span := spans[0]
	for _, v := range spans {
		if span.StartTime >= v.StartTime {
			span = v
		}
	}
	result := []*traceSpan{}
	s.deepSortSpan(span, spans, &result)
	return result
}

func (s *traceConsole) deepSortSpan(span *traceSpan, spans []*traceSpan, result *[]*traceSpan) {
	*result = append(*result, span)
	childs := s.findTraceSpanByParent(spans, span.TraceId)
	sort.Slice(childs, func(i, j int) bool {
		return childs[i].StartTime < childs[j].StartTime
	})
	if len(childs) != 0 {
		for _, v := range childs {
			s.deepSortSpan(v, spans, result)
		}
	}
}

func (s *traceConsole) drawTableTop() {
	color.New(color.FgWhite).Add(color.Underline).Println("┌" + r("─", s.Width-2) + "┐")
	// fmt.Println("┌" + r("─", s.Width-2) + "┐")
}

func (s *traceConsole) drawHorizonalLine() {
	color.New(color.FgWhite).Add(color.Underline).Println("├" + r("─", s.Width-2) + "┤")
	// fmt.Println("├" + r("─", s.Width-2) + "┤")
}

func (s *traceConsole) drawLine(text string, drawError bool) {
	color.New(color.FgWhite).Add(color.Underline).Print("│ ")
	if drawError {
		color.New(color.FgRed).Add(color.Underline).Print(text)
	} else {
		color.New(color.FgWhite).Add(color.Underline).Print(text)
	}
	color.New(color.FgWhite).Add(color.Underline).Println(" │")
	// fmt.Println("│ " + text + " │")
}

func (s *traceConsole) drawTableBottom() {
	color.New(color.FgWhite).Add(color.Underline).Println("└" + r("─", s.Width-2) + "┘")
	// fmt.Println("└" + r("─", s.Width-2) + "┘")
}

func (s *traceConsole) getAlignedTexts(str string, space int) string {
	len := len(str)

	var left string
	if len <= space {
		left = str + r(" ", space-len)
	} else {
		left = str[0:int(math.Max(float64(space-3), 0))]
		left += r(".", int(math.Min(3, float64(space))))
	}

	return left
}

func (s *traceConsole) drawGauge(gstart int, gstop int) string {
	gw := s.GaugeWidth
	p1 := int(gw * gstart / 100)
	p2 := math.Max(float64(int(gw*gstop/100)-p1), 1)
	p3 := math.Max(float64(gw)-(float64(p1)+p2), 0)

	return "[" + r(".", p1) + r("■", int(p2)) + r(".", int(p3)) + "]"
}

func (s *traceConsole) drawSpan(spans []*traceSpan, span *traceSpan, index int, startTime float64, duration float64) {

	startGauge := int(((float64(span.StartTime) - startTime) / duration) * 100)
	endGauge := int(((float64(span.FinishTime) - startTime) / duration) * 100)

	width := s.Width - s.GaugeWidth
	g := s.drawGauge(startGauge, endGauge)

	indent := s.getSpanIndent(spans, span)

	time := fmt.Sprintf("%d", span.Duration/int64(time.Millisecond)) + "ms"
	name := span.Name
	result := ""

	hasError := false
	if span.Error != nil {
		hasError = true
		result = indent + name + " X" + r(" ", width-len(time)-len(indent)-len(name)) + time
	} else {
		if indent == "" {
			result = indent + name + r(" ", width-len(time)-len(indent)-len(name)-6) + time
		} else {
			result = indent + name + r(" ", width-len(time)-len(indent)-len(name)+2) + time
		}
		hasError = false
	}

	s.drawLine(result+g, hasError)

	if len(spans)-1 <= index {
		return
	}
	s.drawSpan(spans, spans[index+1], index+1, startTime, duration)
}

func (s *traceConsole) getSpanIndent(spans []*traceSpan, span *traceSpan) string {

	if span.Tags.CallingLevel > 1 {
		indent := ""
		calcP := s.calcTraceParent(spans, span.ParentId, &[]PrintSpanParent{})
		sort.Slice(calcP, func(i, j int) bool {
			return calcP[i].Callinglevel < calcP[j].Callinglevel
		})
		for _, v := range calcP {
			// fmt.Println(v.Name, ": ", v.Callinglevel, v.Last)
			if v.Last {
				indent += "  "
			} else {
				indent += "| "
			}
		}

		if s.checkLastSpan(spans, span.TraceId, span.ParentId) {
			indent += "└─"
		} else {
			indent += "├─"
		}

		if s.checkHasChild(spans, span.TraceId) {
			indent += "┬─"
		} else {
			indent += "──"
		}

		return indent + " "
	} else {
		return ""
	}
}

func (s *traceConsole) checkLastSpan(spans []*traceSpan, spanId string, parentId string) bool {
	index := 0
	for i, v := range spans {
		if v.TraceId == spanId {
			index = i
		}
	}
	for i := index + 1; i < len(spans); i++ {
		if spans[i].ParentId == parentId {
			return false
		}
	}
	return true
}

func (s traceConsole) checkHasChild(spans []*traceSpan, spanId string) bool {
	for _, v := range spans {
		if v.ParentId == spanId {
			return true
		}
	}
	return false
}

func (s *traceConsole) calcTraceParent(spans []*traceSpan, parentId string, result *[]PrintSpanParent) []PrintSpanParent {

	for _, v := range spans {
		if v.TraceId == parentId {
			if v.ParentId != "" {
				*result = append(*result, PrintSpanParent{
					Name:         v.Name,
					Traceid:      v.TraceId,
					Callinglevel: v.Tags.CallingLevel,
					Last:         s.checkLastSpan(spans, v.TraceId, v.ParentId),
				})
				s.calcTraceParent(spans, v.ParentId, result)
			} else {
				*result = append(*result, PrintSpanParent{
					Name:         v.Name,
					Traceid:      v.TraceId,
					Callinglevel: v.Tags.CallingLevel,
					Last:         true,
				})
			}
		}
	}
	return *result
}

func (s *traceConsole) findTraceSpanByParent(spans []*traceSpan, parentid string) []*traceSpan {
	result := []*traceSpan{}
	for _, v := range spans {
		if v.ParentId == parentid {
			result = append(result, v)
		}
	}
	return result
}

func r(char string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += char
	}
	return result
}
