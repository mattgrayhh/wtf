#!/bin/sh
# Prints the album art of the track playing in Spotify or Apple Music, for
# WTF's image module:
#
#   art:
#     type: image
#     cmd: /path/to/now-playing-artwork.sh
#     refreshInterval: 5
#
# Spotify gives a URL; Music gives raw image data, saved to a temp file.
# Prints nothing when neither app is playing. macOS only.

running() { pgrep -xq "$1"; }

if running Spotify; then
	url=$(osascript -e 'tell application "Spotify" to if player state is playing then return artwork url of current track' 2>/dev/null)
	if [ -n "$url" ]; then
		echo "$url"
		exit 0
	fi
fi

if running Music; then
	out="${TMPDIR:-/tmp}/wtf-now-playing-artwork"
	osascript >/dev/null 2>&1 <<APPLESCRIPT
tell application "Music"
	if player state is not playing then error "not playing"
	set art to first artwork of current track
	set f to open for access POSIX file "$out" with write permission
	set eof f to 0
	write (raw data of art) to f
	close access f
end tell
APPLESCRIPT
	if [ $? -eq 0 ] && [ -s "$out" ]; then
		echo "$out"
		exit 0
	fi
fi
