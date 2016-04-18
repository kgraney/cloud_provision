package dag

import (
	logtest "github.com/Sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

var logger, loghook = logtest.NewNullLogger()

var linearGraph = []Task{
	{
		Name:     "t1",
		Provides: []string{"o1"},
		Action: func(map[string]interface{}) ([]Artifact, error) {
			return []Artifact{
				{Name: "o1", Value: "foobar1"},
			}, nil
		},
	},
	{
		Name:     "t2",
		Provides: []string{"o2"},
		Consumes: []string{"o1"},
		Action: func(input map[string]interface{}) ([]Artifact, error) {
			logger.Info("The value of o1 in t2 is ", input["o1"])
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
		Action: func(input map[string]interface{}) ([]Artifact, error) {
			logger.Info("The value of o1 in t2 is ", input["o1"])
			logger.Info("The value of o2 in t2 is ", input["o2"])
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

func TestToplogicalSortDiamond(t *testing.T) {
	lst := TopologicalSort(diamondGraph)
	assert.Equal(t, 2, len(lst))
	assert.Equal(t, "t1", lst[0].Name)
	assert.Equal(t, "t2", lst[1].Name)
}

func TestToplogicalSortLinear(t *testing.T) {
	lst := TopologicalSort(linearGraph)
	assert.Equal(t, 2, len(lst))
	assert.Equal(t, "t1", lst[0].Name)
	assert.Equal(t, "t2", lst[1].Name)
}

func TestExecuteLinearTasks(t *testing.T) {
	loghook.Reset()
	executor := NewTaskExecutor()
	executor.ExecuteTasks(linearGraph, nil)
	assert.Equal(t, 1, len(loghook.Entries))
	assert.Equal(t, "The value of o1 in t2 is foobar1",
		loghook.LastEntry().Message)
	executor.LogArtifacts()
}

func TestExecuteDiamondTasks(t *testing.T) {
	loghook.Reset()
	executor := NewTaskExecutor()
	executor.ExecuteTasks(diamondGraph, nil)
	assert.Equal(t, 2, len(loghook.Entries))
	assert.Equal(t, "The value of o2 in t2 is foobar2",
		loghook.LastEntry().Message)
	executor.LogArtifacts()
}
