package dag

import (
	"github.com/stretchr/testify/assert"
	"testing"
	//log "github.com/Sirupsen/logrus"
)

var linearGraph = []Task{
	{
		Name:     "t1",
		Provides: []string{"o1"},
		Action: func(map[string]interface{}) ([]Artifact, error) {
			return []Artifact{
				{Name: "o1", Value: "foobar"},
			}, nil
		},
	},
	{
		Name:     "t2",
		Provides: []string{"o2"},
		Consumes: []string{"o1"},
		Action: func(map[string]interface{}) ([]Artifact, error) {
			return []Artifact{
				{Name: "o2", Value: "foobar2"},
			}, nil
		},
	},
}

var diamondGraph = []Task{
	{
		Name:     "t1",
		Provides: []string{"o1", "o2"},
		Action: func(map[string]interface{}) ([]Artifact, error) {
			return []Artifact{
				{Name: "o1", Value: "foobar1"},
				{Name: "o2", Value: "foobar2"},
			}, nil
		},
	},
	{
		Name:     "t2",
		Consumes: []string{"o1", "o2"},
		Action: func(map[string]interface{}) ([]Artifact, error) {
			return []Artifact{}, nil
		},
	},
}

func TestBuildAdjacencyListLinear(t *testing.T) {
	adjList := BuildAdjacencyList(linearGraph)
	assert.Equal(t, 3, len(adjList))

	assert.Equal(t, []string{"o1"}, adjList["t1"])
	assert.Equal(t, []string{"o2"}, adjList["t2"])
	assert.Equal(t, []string{"t2"}, adjList["o1"])
}

func TestBuildAdjacencyListDiamond(t *testing.T) {
	adjList := BuildAdjacencyList(diamondGraph)
	assert.Equal(t, 3, len(adjList))

	assert.Equal(t, []string{"o1", "o2"}, adjList["t1"])
	assert.Equal(t, []string{"t2"}, adjList["o1"])
	assert.Equal(t, []string{"t2"}, adjList["o2"])
}

func TestHasIncomingEdges(t *testing.T) {
	adj := BuildAdjacencyList(diamondGraph)
	assert.Equal(t, false, HasIncomingEdges("t1", adj))
	assert.Equal(t, true, HasIncomingEdges("t2", adj))
	assert.Equal(t, true, HasIncomingEdges("o1", adj))
	assert.Equal(t, true, HasIncomingEdges("o2", adj))
}

func TestToplogicalSort(t *testing.T) {
	lst := TopologicalSort(diamondGraph)
	assert.Equal(t, 4, len(lst))
	assert.Equal(t, "t1", lst[0].Name)
	assert.Equal(t, "t2", lst[3].Name)

}

func TestExecuteTasks(*testing.T) {
	executor := NewTaskExecutor()
	executor.ExecuteTasks(linearGraph, nil)
	executor.LogArtifacts()
}
