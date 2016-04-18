package dag

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
)

// Task names and artifact names must be unique among the entire set of names.

// Tasks require and produce artifacts
type Task struct {
	Name     string
	Consumes []string
	Provides []string
	Action   func(map[string]interface{}) ([]Artifact, error)
}

// Artifacts are consumed and provided for by tasks
type Artifact struct {
	Name  string
	Value interface{}
}

func NewTaskExecutor() *taskExecutor {
	executor := &taskExecutor{}
	executor.artifacts = make(map[string]interface{})
	return executor
}

type taskExecutor struct {
	artifacts map[string]interface{}
}

// Execute the graph and store the artifact results in our map.
func (e *taskExecutor) executeTask(task Task) {
	log.Info(fmt.Sprintf("Executing task [%s]", task.Name))

	task_artifacts := make(map[string]interface{})
	for _, artifact_name := range task.Consumes {
		task_artifacts[artifact_name] = e.artifacts[artifact_name]
	}

	artifacts, err := task.Action(task_artifacts)
	if err != nil {
		log.Warn("Error: ", err)
	}

	for _, artifact := range artifacts {
		e.artifacts[artifact.Name] = artifact.Value
	}
}

func (e *taskExecutor) ExecuteTasks(tasks []Task, artifacts []Artifact) {
	for _, task := range TopologicalSort(tasks) {
		e.executeTask(task)
	}
}

func (e *taskExecutor) LogArtifacts() {
	for name, value := range e.artifacts {
		log.Info("Artifact: ", name, value)
	}
}

// Build an adjacency list representation of the DAG for a given list of tasks.
//    Edges represented go from the list key to its dependents.
func BuildAdjacencyList(tasks []Task) map[string]([]string) {

	dag := map[string]([]string){}
	for _, task := range tasks {
		for _, node := range task.Provides {
			dag[task.Name] = append(dag[task.Name], node)
		}
		for _, node := range task.Consumes {
			dag[node] = append(dag[node], task.Name)
		}
		// TODO(kmg): add the node if it's not already in the list
		// in case it's not referenced anywhere else
	}

	return dag
}

func HasIncomingEdges(name string, adj map[string]([]string)) bool {
	for _, tlist := range adj {
		for _, t := range tlist {
			if t == name {
				return true
			}
		}
	}
	return false
}

// Kahn's algorithm
func TopologicalSort(tasks []Task) []Task {
	order := topologicalSort(tasks)

	// Convert the list of task/artifact names into a list of task objects
	m := make(map[string]Task)
	for _, t := range tasks {
		m[t.Name] = t
	}

	lst := []Task{}
	for _, s := range order {
		if task, ok := m[s]; ok {
			lst = append(lst, task)
		}
	}

	return lst
}

// Finds a topological ordering of task AND artifact names
func topologicalSort(tasks []Task) []string {
	adjLst := BuildAdjacencyList(tasks)

	lst := []string{}
	set := make(map[string]bool)
	for _, t := range tasks {
		if !HasIncomingEdges(t.Name, adjLst) {
			set[t.Name] = true
		}
	}

	for len(set) > 0 {
		var n string
		for n, _ = range set {
			break
		}
		delete(set, n)
		lst = append(lst, n)
		for len(adjLst[n]) > 0 {
			for i, m := range adjLst[n] {
				adjLst[n][i] = adjLst[n][len(adjLst[n])-1]
				adjLst[n] = adjLst[n][:len(adjLst[n])-1]
				if !HasIncomingEdges(m, adjLst) {
					set[m] = true
				}
				break
			}
		}
	}

	return lst
}
