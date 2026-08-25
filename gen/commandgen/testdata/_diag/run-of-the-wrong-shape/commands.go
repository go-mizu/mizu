package broken

//mizu:command name=go
type Command struct{}

func (c *Command) Run() error { return nil }
