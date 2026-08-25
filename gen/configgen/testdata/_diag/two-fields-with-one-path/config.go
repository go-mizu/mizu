package broken

//mizu:config
type Config struct {
	App struct {
		Name string
		Also string `toml:"name"`
	}
}
