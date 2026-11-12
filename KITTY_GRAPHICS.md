# Images and charts

This fork adds three modules that draw real images in terminals supporting
the [kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
(kitty, Ghostty, WezTerm, Rex and others). In other terminals they fall back
to tview's coloured block characters.

See [`_sample_configs/kitty_graphics_config.yml`](_sample_configs/kitty_graphics_config.yml).

## `image`

Shows a picture from one of:

| Key | Meaning |
|---|---|
| `path` | A local PNG, JPEG, GIF or WebP file |
| `url` | An image URL, fetched on every refresh (e.g. `https://wttr.in/London_0pq.png`) |
| `cmd`, `args` | A command that prints image data, or a path or URL to one |
| `fit` | `contain` (default) keeps the image's shape; `fill` stretches it |

`scripts/now-playing-artwork.sh` prints the album art for whatever Spotify or
Apple Music is playing, for use as an `image` module's `cmd`.

## `chart`

Draws `line`, `area` or `bar` charts (`kind`) from CSV, TSV or
space-separated data read from `cmd`/`args`, `path` or `url`:

```
month,Revenue,Costs     <- optional header names the series
Jan,120,90              <- optional first-column labels
Feb,135,                <- empty cells leave gaps
```

A single column of numbers is fine too. Other keys: `colors` (hex list),
`secondary` (series names to plot on a right-hand axis), `legend`
(`auto`/`true`/`false`), `maxPoints`, `yMin`, `yMax`, `yFormat` (e.g.
`"%.0f°"`), `textColor`, `fontSize`.

## `weatherchart`

Hourly temperature and chance of rain from [Open-Meteo](https://open-meteo.com/)
(no API key). Set `city`, or `latitude` and `longitude`; plus `units`
(`metric`/`imperial`) and `hours` (6–168). The panel title shows the current
temperature, conditions and today's high and low.

## `wunderground`

A weather card: current conditions from a Weather Underground personal
weather station beside a 3–7 day outlook with icons and temperature-range
bars. Keys: `stationId`, `apiKey` (free for station owners at
wunderground.com/member/api-keys), `latitude`/`longitude`, `place`, `units`,
`days`, `timezone`. Without an `apiKey` it uses Open-Meteo at the given
coordinates, so the card works before you have a key.

## `bigclock`

A large clock drawn as an image, so it fills any panel. Keys: `timezone`,
`hourFormat` (`12`/`24`), `dateFormat` (Go layout), `color`.

## Other additions

- **`outlookcal`**: today's and upcoming Outlook events through the
  [Composio](https://composio.dev) CLI (`composio execute
  OUTLOOK_GET_CALENDAR_VIEW`). Past events dim, the current one is green, the
  next shows a countdown; canceled and declined events are hidden. Keys:
  `account`, `days`, `timezone`, `hourFormat`, `fetchInterval`,
  `showCanceled`, `showDeclined`.
- **`todoist`**: new `filters` (a list of queries, or `{name, query}` maps)
  shown as projects, sorted by priority then due date, with priority dots and
  due labels; `limit` caps the list. `stacked: true` shows every project
  and filter at once under its own heading (selection and closing tasks
  work across all of them) instead of one at a time.
- **`digitalclock`**: new `timezone` setting.
- **Keychain secrets**: any `apiKey` can be `keychain:SERVICE/ACCOUNT` to
  read a macOS Keychain generic password, e.g. one stored with
  `security add-generic-password -s glass -a todoist -w`.

## How it works

`kittygfx` asks the terminal at startup whether it understands the protocol,
then each image panel:

1. uploads its PNG once with an image ID and a *virtual placement* sized to
   the panel (`a=t` then `a=p,U=1`), and
2. fills its cells with the Unicode placeholder `U+10EEEE`, the image ID in
   the foreground colour and the row and column as combining diacritics.

To tcell the placeholders are ordinary text, so redraws, resizing and help
modals drawn over an image all just work. Uploads are queued and written in
tview's after-draw hook, on the draw goroutine, so they never interleave with
tcell's output. Charts are re-rendered at the panel's exact pixel size, with
2x supersampling when the terminal reports small (HiDPI point-sized) cells.

Set `wtf.kittyGraphics` to `true` to skip detection (for multiplexers that
pass graphics through without answering the query) or `false` to turn images
off; the `WTF_KITTY_GRAPHICS` environment variable overrides both.
