package bot

import (
	"bytes"
	_ "embed"
	html "html/template"
	"strings"
	text "text/template"

	"github.com/Masterminds/sprig/v3"

	"gitlab.com/slxh/matrix/alertmanager_matrix/internal/util"
	"gitlab.com/slxh/matrix/alertmanager_matrix/pkg/alertmanager"
)

// Default alert template values.
const (
	DefaultTextTemplate = `{{ range .Alerts }}{{.StatusString|icon}} {{.StatusString|upper}} {{.AlertName}}: {{.Summary}}{{if ne .Fingerprint ""}} ({{.Fingerprint}}){{end}}{{if $.ShowLabels}}, labels: {{.LabelString}}{{end}}\n{{ end -}}`                                                                                //nolint:lll
	DefaultHTMLTemplate = `{{ range .Alerts }}<font color="{{.StatusString|color}}">{{.StatusString|icon}} <b>{{.StatusString|upper}}</b> {{.AlertName}}:</font> {{.Summary}}{{if ne .Fingerprint ""}} ({{.Fingerprint}}){{end}}{{if $.ShowLabels}}<br/><b>Labels:</b> <code>{{.LabelString}}</code>{{end}}<br/>{{- end -}}` //nolint:lll
)

//go:embed templates/silence.md.tmpl
var silenceTemplate string

// Default color and icon values.
var (
	DefaultColors = map[string]string{ //nolint:gochecknoglobals
		"alert":       "black",
		"information": "blue",
		"info":        "blue",
		"warning":     "orange",
		"critical":    "red",
		"error":       "red",
		"resolved":    "green",
		"silenced":    "gray",
	}

	DefaultIcons = map[string]string{ //nolint:gochecknoglobals
		"alert":       "🔔️",
		"information": "ℹ️",
		"info":        "ℹ️",
		"warning":     "⚠️",
		"critical":    "🚨",
		"error":       "🚨",
		"resolved":    "✅",
		"silenced":    "🔕",
	}
)

// Formatter represents a NewMessage formatter with an icon and color set.
type Formatter struct {
	colors  map[string]string
	icons   map[string]string
	text    *text.Template
	html    *html.Template
	silence *text.Template
}

// NewFormatter creates a new formatter with the given text/HTML templates, colors and strings.
// The default templates, colors or icons are used if "" or nil is provided.
//
// The following functions are registered for use in the templates:
//
//	icon:  returns the icon for the given string.
//	color: returns the color for the given string.
//	upper: converts the given string to uppercase.
//	lower: converts the given string to lowercase.
//	title: converts the given string to title case.
func NewFormatter(textTemplate, htmlTemplate string, colors, icons map[string]string) *Formatter {
	if textTemplate == "" {
		textTemplate = DefaultTextTemplate
	}

	if htmlTemplate == "" {
		htmlTemplate = DefaultHTMLTemplate
	}

	if colors == nil {
		colors = DefaultColors
	}

	if icons == nil {
		icons = DefaultIcons
	}

	f := &Formatter{colors: colors, icons: icons}
	funcMap := map[string]interface{}{
		"icon":  f.icon,
		"color": f.color,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.ToTitle,
		"deref": util.ValueOrDefault[string],
	}
	f.text = text.Must(text.New("").Funcs(sprig.FuncMap()).Funcs(funcMap).Parse(textTemplate))
	f.html = html.Must(html.New("").Funcs(sprig.FuncMap()).Funcs(funcMap).Parse(htmlTemplate))
	f.silence = text.Must(text.New("").Funcs(sprig.FuncMap()).Funcs(funcMap).Parse(silenceTemplate))

	return f
}

// icon returns the icon for a string.
func (f *Formatter) icon(t string) string {
	if e, ok := f.icons[t]; ok {
		return e
	}

	return "❔"
}

// color returns the color for string.
func (f *Formatter) color(t string) string {
	if c, ok := f.colors[t]; ok {
		return c
	}

	return "gray"
}

// FormatAlerts formats alerts as plain text and HTML.
func (f *Formatter) FormatAlerts(alerts []*alertmanager.Alert, labels bool) (string, string) {
	var plain, html strings.Builder

	message := &Message{Alerts: alerts, ShowLabels: labels}

	if err := f.text.Execute(&plain, message); err != nil {
		return err.Error(), err.Error()
	}

	if err := f.html.Execute(&html, message); err != nil {
		return err.Error(), err.Error()
	}

	return plain.String(), html.String()
}

// FormatSilences formats silences as Markdown.
func (f *Formatter) FormatSilences(silences []alertmanager.Silence, state string) (md string) {
	buf := &bytes.Buffer{}
	filtered := make([]alertmanager.Silence, 0, len(silences))

	for _, s := range silences {
		if s.Status() == state {
			filtered = append(filtered, s)
		}
	}

	if err := f.silence.Execute(buf, filtered); err != nil {
		return err.Error()
	}

	return buf.String()
}
