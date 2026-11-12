package todo_plus

import (
	"fmt"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/modules/todo_plus/backend"
	"github.com/wtfutil/wtf/utils"
)

func (widget *Widget) content() (string, string, bool) {
	if widget.settings.stacked {
		return widget.stackedContent()
	}

	proj := widget.CurrentProject()

	if proj == nil {
		return widget.CommonSettings().Title, "", false
	}

	if proj.Err != nil {
		return widget.CommonSettings().Title, proj.Err.Error(), true
	}

	title := fmt.Sprintf(
		"[%s]%s[white]",
		widget.settings.Colors.Title,
		proj.Name)

	str := ""

	for idx, item := range proj.Tasks {
		str += widget.taskRow(item, idx)
	}
	return title, str, false
}

// stackedContent lists every project under its own heading.
func (widget *Widget) stackedContent() (string, string, bool) {
	title := widget.CommonSettings().Title
	str := ""
	row := 0

	for i, proj := range widget.projects {
		if i > 0 {
			str += "\n"
		}
		count := ""
		if n := len(proj.Tasks); n > 0 {
			count = fmt.Sprintf(" [gray]%d[-]", n)
		}
		str += fmt.Sprintf("[%s::b]%s[-::-]%s\n", widget.settings.Colors.Subheading, tview.Escape(proj.Name), count)

		switch {
		case proj.Err != nil:
			str += "  [red]" + tview.Escape(proj.Err.Error()) + "[-]\n"
		case len(proj.Tasks) == 0:
			str += "  [gray]Nothing here[-]\n"
		}
		for _, item := range proj.Tasks {
			str += widget.taskRow(item, row)
			row++
		}
	}

	return title, str, false
}

// taskRow renders one selectable task: priority dot, name and due date.
func (widget *Widget) taskRow(item backend.Task, idx int) string {
	rowColor := widget.RowColor(idx)

	marker := "| |"
	if item.Priority > 0 {
		marker = fmt.Sprintf("[%s]%s[%s]", priorityColor(item.Priority), priorityMarker, rowColor)
	}

	due, dueLen := "", 0
	if item.Due != "" {
		dueColor := "gray"
		if item.Overdue {
			dueColor = "red"
		}
		due = fmt.Sprintf("  [%s]%s[%s]", dueColor, tview.Escape(item.Due), rowColor)
		dueLen = 2 + len([]rune(item.Due))
	}

	row := fmt.Sprintf(`[%s]%s %s%s[%s]`, rowColor, marker, tview.Escape(item.Name), due, rowColor)
	return utils.HighlightableHelper(widget.View, row, idx, len([]rune(item.Name))+dueLen)
}

const priorityMarker = "●"

// priorityColor follows Todoist's own colours for p1 to p4.
func priorityColor(priority int) string {
	switch priority {
	case 1:
		return "#ff5c5c"
	case 2:
		return "#ffa04d"
	case 3:
		return "#5c9dff"
	default:
		return "#606060"
	}
}

func (widget *Widget) display() {
	widget.Redraw(widget.content)
}
