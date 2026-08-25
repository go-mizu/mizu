package broken

//mizu:config
type Config struct {
	App struct {
		Name   string
		Prefix string `default:"{App.Name}:"`
	}
}
