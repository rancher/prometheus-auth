package view

type vertex struct {
	value        string
	adjacencyMap map[string]*vertex
}

func (o *vertex) addAdjacency(n *vertex) {
	if o == nil || n == nil {
		return
	}

	o.adjacencyMap[n.value] = n
}

func (o *vertex) delAdjacency(n *vertex) {
	if o == nil || n == nil {
		return
	}

	delete(o.adjacencyMap, n.value)
}

func (o *vertex) getAdjacencyList() []*vertex {
	if o == nil {
		return make([]*vertex, 0)
	}

	vertexes := make([]*vertex, 0, len(o.adjacencyMap))

	for _, adjacency := range o.adjacencyMap {
		vertexes = append(vertexes, adjacency)
	}

	return vertexes
}

func (o *vertex) isAdjacency(n *vertex) bool {
	if o == nil || n == nil {
		return false
	}

	_, exist := o.adjacencyMap[n.value]

	return exist
}

func newVertex(value string) *vertex {
	return &vertex{
		value:        value,
		adjacencyMap: make(map[string]*vertex, 4),
	}
}

type edgeDirection int

const (
	directedGraph edgeDirection = iota
	undirectedGraph
)

type graph struct {
	vertexes  map[string]*vertex
	direction edgeDirection
}

func (o *graph) addEdge(source, destination string) (sourceVertex, destinationVertex *vertex) {
	if o == nil {
		return
	}

	sourceVertex = o.addVertex(source)
	destinationVertex = o.addVertex(destination)

	sourceVertex.addAdjacency(destinationVertex)
	if o.direction == undirectedGraph {
		destinationVertex.addAdjacency(sourceVertex)
	}

	return
}

func (o *graph) delEdge(source, destination string) (sourceVertex, destinationVertex *vertex) {
	if o == nil {
		return
	}

	sourceVertex = o.delVertex(source)
	destinationVertex = o.delVertex(destination)

	if sourceVertex != nil && destinationVertex != nil {
		sourceVertex.delAdjacency(destinationVertex)
		if o.direction == undirectedGraph {
			destinationVertex.delAdjacency(sourceVertex)
		}
	}

	return
}

func (o *graph) addVertex(value string) *vertex {
	if o == nil {
		return nil
	}

	ret, exist := o.vertexes[value]

	if !exist {
		ret = newVertex(value)
		o.vertexes[value] = ret
	}

	return ret
}

func (o *graph) delVertex(value string) *vertex {
	if o == nil {
		return nil
	}

	ret, exist := o.vertexes[value]

	if exist {
		for val, node := range o.vertexes {
			if val != value {
				node.delAdjacency(ret)
			}
		}

		delete(o.vertexes, value)
	}
	return ret
}

func (o *graph) deepCopy() *graph {
	if o == nil {
		return nil
	}

	ret := *o

	return &ret
}

func newGraph(direction edgeDirection) *graph {
	return &graph{
		vertexes:  make(map[string]*vertex, 8),
		direction: direction,
	}
}
