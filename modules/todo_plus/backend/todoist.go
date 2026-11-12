package backend

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gopherlibs/todoist/api"
	"github.com/olebedev/config"
)

type Todoist struct {
	client     *api.Client
	httpClient *http.Client
	apiKey     string
	projects   []interface{}
	filters    []todoistFilter
	limit      int
}

func (todo *Todoist) Title() string {
	return "Todoist"
}

func (todo *Todoist) Setup(config *config.Config) {

	todo.apiKey = config.UString("apiKey")
	todo.client = api.New(todo.apiKey)
	todo.httpClient = &http.Client{Timeout: 20 * time.Second}
	todo.projects = config.UList("projects")
	todo.filters = parseFilters(config.UList("filters"))
	todo.limit = config.UInt("limit", 0)
}

func (todo *Todoist) BuildProjects() []*Project {
	projects := []*Project{}

	for i := range todo.filters {
		projects = append(projects, todo.GetProject(fmt.Sprintf("%s%d", filterPrefix, i)))
	}

	for _, id := range todo.projects {
		i := fmt.Sprintf("%v", id)
		proj := todo.GetProject(i)
		projects = append(projects, proj)
	}
	return projects
}

func (todo *Todoist) GetProject(id string) *Project {
	// Todoist seems to experience a lot of network issues on their side
	// If we can't connect, handle it with an empty project until we can
	proj := &Project{
		Index:   -1,
		backend: todo,
	}

	proj.ID = id
	proj.Name = "Error"

	if i, ok := filterIndex(id); ok && i < len(todo.filters) {
		proj.Name = todo.filters[i].Name
		proj.Tasks, proj.Err = todo.LoadTasks(id)
		return proj
	}

	p, err := todo.client.Project(id)
	if err != nil {
		return proj
	}

	proj.Name = p.Name

	tasks, err := todo.LoadTasks(proj.ID)
	proj.Err = err
	proj.Tasks = tasks

	return proj
}

func toTask(task api.Task) Task {
	return Task{
		ID:        task.ID,
		Completed: task.Checked,
		Name:      task.Content,
	}
}

func (todo *Todoist) LoadTasks(id string) ([]Task, error) {
	if i, ok := filterIndex(id); ok && i < len(todo.filters) {
		return todo.filterTasks(todo.filters[i].Query, todo.limit)
	}

	tasks, err := todo.client.Tasks(id)
	if err != nil {
		return nil, err
	}

	var finalTasks []Task
	for _, item := range tasks.Results {
		finalTasks = append(finalTasks, toTask(item))
	}
	return finalTasks, nil
}

func (todo *Todoist) CloseTask(task *Task) error {
	if task != nil {
		_, err := todo.client.TaskClose(task.ID)
		return err
	}
	return nil
}

func (todo *Todoist) DeleteTask(task *Task) error {
	if task != nil {
		_, err := todo.client.TaskDelete(task.ID)
		return err
	}
	return nil
}

func (todo *Todoist) Sources() []string {
	var result []string
	for i := range todo.filters {
		result = append(result, fmt.Sprintf("%s%d", filterPrefix, i))
	}
	for _, id := range todo.projects {
		i := fmt.Sprintf("%v", id)
		result = append(result, i)
	}
	return result
}
