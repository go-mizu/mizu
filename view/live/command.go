package live

// Command is a client-side action sent from server to client.
type Command interface {
	commandType() string
}

// Redirect navigates the browser to a new URL.
type Redirect struct {
	To      string `json:"to"`
	Replace bool   `json:"replace,omitempty"`
}

func (Redirect) commandType() string { return "redirect" }

// Focus sets focus to an element.
type Focus struct {
	Selector string `json:"selector"`
}

func (Focus) commandType() string { return "focus" }

// Scroll scrolls to an element or position.
type Scroll struct {
	Selector string `json:"selector,omitempty"`
	Block    string `json:"block,omitempty"` // "start", "center", "end", "nearest"
}

func (Scroll) commandType() string { return "scroll" }

// Download triggers a file download.
type Download struct {
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
}

func (Download) commandType() string { return "download" }

// JS executes arbitrary JavaScript.
type JS struct {
	Code string         `json:"code"`
	Args map[string]any `json:"args,omitempty"`
}

func (JS) commandType() string { return "js" }

// SetTitle sets the document title.
type SetTitle struct {
	Title string `json:"title"`
}

func (SetTitle) commandType() string { return "title" }

// AddClass adds CSS class(es) to an element.
type AddClass struct {
	Selector string `json:"selector"`
	Class    string `json:"class"`
}

func (AddClass) commandType() string { return "add_class" }

// RemoveClass removes CSS class(es) from an element.
type RemoveClass struct {
	Selector string `json:"selector"`
	Class    string `json:"class"`
}

func (RemoveClass) commandType() string { return "remove_class" }

// ToggleClass toggles CSS class(es) on an element.
type ToggleClass struct {
	Selector string `json:"selector"`
	Class    string `json:"class"`
}

func (ToggleClass) commandType() string { return "toggle_class" }

// SetAttribute sets an attribute on an element.
type SetAttribute struct {
	Selector string `json:"selector"`
	Name     string `json:"name"`
	Value    string `json:"value"`
}

func (SetAttribute) commandType() string { return "set_attr" }

// RemoveAttribute removes an attribute from an element.
type RemoveAttribute struct {
	Selector string `json:"selector"`
	Name     string `json:"name"`
}

func (RemoveAttribute) commandType() string { return "remove_attr" }

// commandEnvelope wraps a command for wire serialization.
type commandEnvelope struct {
	Cmd  string `json:"cmd"`
	Data any    `json:"data"`
}

func wrapCommand(c Command) commandEnvelope {
	return commandEnvelope{
		Cmd:  c.commandType(),
		Data: c,
	}
}
