package todo_plus

import (
	"log"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/modules/todo_plus/backend"
	"github.com/wtfutil/wtf/view"
)

// A Widget represents a Todoist widget
type Widget struct {
	view.MultiSourceWidget
	view.ScrollableWidget

	projects []*backend.Project
	settings *Settings
	backend  backend.Backend
}

// NewWidget creates a new instance of a widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		MultiSourceWidget: view.NewMultiSourceWidget(settings.Common, "project", "projects"),
		ScrollableWidget:  view.NewScrollableWidget(tviewApp, redrawChan, pages, settings.Common),

		settings: settings,
	}

	widget.backend = getBackend(settings.backendType)
	widget.backend.Setup(settings.backendSettings)
	widget.CommonSettings().Title = widget.backend.Title()

	widget.SetRenderFunction(widget.display)
	widget.initializeKeyboardControls()
	widget.SetDisplayFunction(widget.display)

	return &widget
}

func getBackend(backendType string) backend.Backend {
	switch backendType {
	case "trello":
		backend := &backend.Trello{}
		return backend
	case "todoist":
		backend := &backend.Todoist{}
		return backend
	default:
		log.Fatal(backendType + " is not a supported backend")
		return nil
	}

}

/* -------------------- Exported Functions -------------------- */

func (widget *Widget) CurrentProject() *backend.Project {
	return widget.ProjectAt(widget.Idx)
}

func (widget *Widget) ProjectAt(idx int) *backend.Project {
	if len(widget.projects) == 0 {
		return nil
	}

	return widget.projects[idx]
}

func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	widget.projects = widget.backend.BuildProjects()
	widget.Sources = widget.backend.Sources()
	widget.updateItemCount()
	widget.display()
}

// taskRef locates one row of the stacked view.
type taskRef struct {
	project *backend.Project
	index   int
}

// rows lists every task in stacked order.
func (widget *Widget) rows() []taskRef {
	var refs []taskRef
	for _, p := range widget.projects {
		for i := range p.Tasks {
			refs = append(refs, taskRef{p, i})
		}
	}
	return refs
}

func (widget *Widget) updateItemCount() {
	if widget.settings.stacked {
		widget.SetItemCount(len(widget.rows()))
		return
	}
	if p := widget.CurrentProject(); p != nil {
		widget.SetItemCount(len(p.Tasks))
	}
}

// selectedRef returns the project and task under the cursor in stacked view.
func (widget *Widget) selectedRef() (taskRef, bool) {
	rows := widget.rows()
	if widget.Selected < 0 || widget.Selected >= len(rows) {
		return taskRef{}, false
	}
	return rows[widget.Selected], true
}

func (widget *Widget) NextSource() {
	if widget.settings.stacked {
		return
	}
	widget.MultiSourceWidget.NextSource()
	widget.Selected = widget.CurrentProject().Index
	widget.SetItemCount(len(widget.CurrentProject().Tasks))
	widget.RenderFunction()
}

func (widget *Widget) PrevSource() {
	if widget.settings.stacked {
		return
	}
	widget.MultiSourceWidget.PrevSource()
	widget.Selected = widget.CurrentProject().Index
	widget.SetItemCount(len(widget.CurrentProject().Tasks))
	widget.RenderFunction()
}

func (widget *Widget) Prev() {
	widget.ScrollableWidget.Prev()
	if !widget.settings.stacked {
		widget.CurrentProject().Index = widget.Selected
	}
}

func (widget *Widget) Next() {
	widget.ScrollableWidget.Next()
	if !widget.settings.stacked {
		widget.CurrentProject().Index = widget.Selected
	}
}

func (widget *Widget) Unselect() {
	widget.ScrollableWidget.Unselect()
	if !widget.settings.stacked {
		widget.CurrentProject().Index = -1
	}
	widget.RenderFunction()
}

// stackedAction runs a task action on the selected row in stacked view.
func (w *Widget) stackedAction(act func(p *backend.Project)) {
	ref, ok := w.selectedRef()
	if !ok {
		return
	}
	ref.project.Index = ref.index
	act(ref.project)
	ref.project.Index = -1

	w.updateItemCount()
	if w.Selected >= len(w.rows()) {
		w.Prev()
	}
	w.RenderFunction()
}

/* -------------------- Keyboard Movement -------------------- */

// Close closes the currently-selected task in the currently-selected project
func (w *Widget) Close() {
	if w.settings.stacked {
		w.stackedAction(func(p *backend.Project) { p.CloseSelectedTask() })
		return
	}
	w.CurrentProject().CloseSelectedTask()
	w.SetItemCount(len(w.CurrentProject().Tasks))

	if w.CurrentProject().IsLast() {
		w.Prev()
		return
	}
	w.CurrentProject().Index = w.Selected
	w.RenderFunction()
}

// Delete deletes the currently-selected task in the currently-selected project
func (w *Widget) Delete() {
	if w.settings.stacked {
		w.stackedAction(func(p *backend.Project) { p.DeleteSelectedTask() })
		return
	}
	w.CurrentProject().DeleteSelectedTask()
	w.SetItemCount(len(w.CurrentProject().Tasks))

	if w.CurrentProject().IsLast() {
		w.Prev()
	}
	w.CurrentProject().Index = w.Selected
	w.RenderFunction()
}
