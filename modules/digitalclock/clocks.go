package digitalclock

import (
	"fmt"
	"strconv"
	"time"
)

// AM defines the AM string format
const AM = "A"

// PM defines the PM string format
const PM = "P"
const minRowsForBorder = 3

// Converts integer to string along with makes sure the length of string is > 2
func intStrConv(val int) string {
	valStr := strconv.Itoa(val)

	if len(valStr) < 2 {
		valStr = "0" + valStr
	}
	return valStr
}

// Returns Hour + minute + AM/PM information based on the settings
func getHourMinute(t time.Time, hourFormat string) string {
	strHours := intStrConv(t.Hour())
	AMPM := " "

	if hourFormat == "12" {
		hour := t.Hour()
		strHours = intStrConv(hour % 12)
		if (hour % 12) == hour {
			AMPM = AM
		} else {
			AMPM = PM
		}

	}

	strMinutes := intStrConv(t.Minute())
	strMinutes += AMPM
	return strHours + getColon(t) + strMinutes
}

// Returns the : with blinking based on the seconds
func getColon(t time.Time) string {
	if t.Second()%2 == 0 {
		return ":"
	}
	return " "
}

func getDate(t time.Time, dateFormat string, withDatePrefix bool) string {
	if withDatePrefix {
		return fmt.Sprintf("Date: %s", t.Format(dateFormat))
	}
	return t.Format(dateFormat)
}

func getUTC(t time.Time) string {
	return fmt.Sprintf("UTC: %s", t.UTC().Format(time.RFC3339))
}

func getEpoch(t time.Time) string {
	return fmt.Sprintf("Epoch: %d", t.Unix())
}

// Renders the clock as string by accessing appropriate font from configured in settings
func renderClock(widgetSettings Settings) (string, bool) {
	var digFont ClockFont
	clockTime := getHourMinute(widgetSettings.now(), widgetSettings.hourFormat)
	digFont = getFont(widgetSettings)

	chars := [][]string{}
	for _, char := range clockTime {
		chars = append(chars, digFont.get(string(char)))
	}

	needBorder := digFont.fontRows <= minRowsForBorder
	return fontsJoin(chars, digFont.fontRows, widgetSettings.color), needBorder
}
