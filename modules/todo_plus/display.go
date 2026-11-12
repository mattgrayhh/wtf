package todo_plus

import (
	"fmt"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/utils"
)

func (widget *Widget) content() (string, string, bool) {
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

		row := fmt.Sprintf(
			`[%s]%s %s%s[%s]`,
			rowColor,
			marker,
			tview.Escape(item.Name),
			due,
			rowColor,
		)

		str += utils.HighlightableHelper(widget.View, row, idx, len([]rune(item.Name))+dueLen)
	}
	return title, str, false
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
