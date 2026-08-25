package broken

//mizu:validate
type Listing struct {
	Down Level1 `json:"down"`
}

type Level1 struct {
	Down Level2 `json:"down"`
}

type Level2 struct {
	Down Level3 `json:"down"`
}

type Level3 struct {
	Down Level4 `json:"down"`
}

type Level4 struct {
	Down Level5 `json:"down"`
}

type Level5 struct {
	Down Level6 `json:"down"`
}

type Level6 struct {
	Down Level7 `json:"down"`
}

type Level7 struct {
	Down Level8 `json:"down"`
}

type Level8 struct {
	Down Level9 `json:"down"`
}

type Level9 struct {
	Down Level10 `json:"down"`
}

type Level10 struct {
	Down Level11 `json:"down"`
}

type Level11 struct {
	Down Level12 `json:"down"`
}

type Level12 struct {
	Down Level13 `json:"down"`
}

type Level13 struct {
	Down Level14 `json:"down"`
}

type Level14 struct {
	Down Level15 `json:"down"`
}

type Level15 struct {
	Q string `json:"q" validate:"required"`
}
