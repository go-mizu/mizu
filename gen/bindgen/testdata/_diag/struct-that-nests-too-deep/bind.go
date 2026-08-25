package broken

//mizu:bind
type Listing struct {
	Down Level1 `form:"down"`
}

type Level1 struct {
	Down Level2 `form:"down"`
}

type Level2 struct {
	Down Level3 `form:"down"`
}

type Level3 struct {
	Down Level4 `form:"down"`
}

type Level4 struct {
	Down Level5 `form:"down"`
}

type Level5 struct {
	Down Level6 `form:"down"`
}

type Level6 struct {
	Down Level7 `form:"down"`
}

type Level7 struct {
	Down Level8 `form:"down"`
}

type Level8 struct {
	Down Level9 `form:"down"`
}

type Level9 struct {
	Down Level10 `form:"down"`
}

type Level10 struct {
	Down Level11 `form:"down"`
}

type Level11 struct {
	Down Level12 `form:"down"`
}

type Level12 struct {
	Down Level13 `form:"down"`
}

type Level13 struct {
	Down Level14 `form:"down"`
}

type Level14 struct {
	Down Level15 `form:"down"`
}

type Level15 struct {
	Q string `form:"q"`
}
