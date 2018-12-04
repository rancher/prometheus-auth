package view

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

type VertexKind struct {
	hashWrap  bool
	decorator string
	quitValue string
}

func (o *VertexKind) Wrap(outValue string) string {
	if o.hashWrap {
		outValue = o.hash(outValue)
	}

	return o.decorator + outValue
}

func (o *VertexKind) Is(inValue string) bool {
	return strings.HasPrefix(inValue, o.decorator)
}

func (o *VertexKind) Quit(intValue string) bool {
	return o.quitValue == o.UnWrap(intValue)
}

// UnWrap can unwrap hashed value
func (o *VertexKind) UnWrap(inValue string) string {
	return strings.TrimPrefix(inValue, o.decorator)
}

func (o *VertexKind) hash(value string) string {
	h := sha1.New()
	h.Write([]byte(value))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func NewVertexKind(decorator, quitValue string, hashWrap bool) *VertexKind {
	v := &VertexKind{
		hashWrap:  hashWrap,
		decorator: decorator,
		quitValue: quitValue,
	}

	if hashWrap {
		v.quitValue = v.hash(v.quitValue)
	}

	return v
}

type GraphSearchResult struct {
	SetView
	quit bool
}

func (o *GraphSearchResult) QuitEarly() bool {
	return o.quit
}

func (o *GraphSearchResult) Values() []string {
	return o.GetAll()
}

func newGraphSearchResult() *GraphSearchResult {
	return &GraphSearchResult{
		SetView: NewSetView(),
	}
}

type VertexOperation func(graph Graph, value string)
type EdgeOperation func(graph Graph, from, to string)
type SearchOperation int

const (
	/**
	SearchOperation
	*/
	DFS SearchOperation = iota
	BFS
)

type Graph interface {
	AddVertex(value string)
	DelVertex(value string)
	AddEdge(from, to string)
	DelEdge(from, to string)
}

type GraphView interface {
	Vertex(op VertexOperation, vertexValue string)
	Edge(op EdgeOperation, from, to string)
	Search(first string, op SearchOperation, resultVertexKind *VertexKind) *GraphSearchResult
}

type graphView struct {
	graph *graph
}

func (o *graphView) AddVertex(value string) {
	o.graph.addVertex(value)
}

func (o *graphView) DelVertex(value string) {
	o.graph.delVertex(value)
}

func (o *graphView) AddEdge(from, to string) {
	o.graph.addEdge(from, to)
}

func (o *graphView) DelEdge(from, to string) {
	o.graph.delEdge(from, to)
}

func (o *graphView) Vertex(op VertexOperation, vertexValue string) {
	if len(vertexValue) == 0 {
		return
	}

	op(o, vertexValue)
}

func (o *graphView) Edge(op EdgeOperation, from, to string) {
	if len(from) == 0 || len(to) == 0 {
		return
	}

	op(o, from, to)
}

func (o *graphView) Search(first string, op SearchOperation, resultVertexKind *VertexKind) *GraphSearchResult {
	ret := newGraphSearchResult()
	visited := make(map[string]bool, 8)
	visitList := make([]*vertex, 0, 8)
	g := o.graph.deepCopy()

	firstVertex := g.vertexes[first]
	if firstVertex == nil {
		return ret
	}

	visitList = append(visitList, firstVertex)
	for {
		if len(visitList) == 0 {
			break
		}

		var access *vertex
		switch op {
		case BFS:
			access = visitList[0]
		case DFS:
			access = visitList[len(visitList)-1]
		}

		if !visited[access.value] {
			if resultVertexKind.Is(access.value) {
				hasAll := resultVertexKind.Quit(access.value)
				if hasAll {
					ret.quit = true
					break
				}

				ret.Put(resultVertexKind.UnWrap(access.value))
			}
			visitList = append(visitList, g.vertexes[access.value].getAdjacencyList()...)
			visited[access.value] = true
		}

		switch op {
		case BFS:
			visitList = visitList[1:]
		case DFS:
			visitList = visitList[0 : len(visitList)-1]
		}
	}

	return ret
}

func NewGraphView() GraphView {
	return &graphView{
		graph: newGraph(directedGraph),
	}
}
