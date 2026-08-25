// This module exists to give archtest an exported API with a known shape. It is
// under testdata so the go command leaves it out of the toolkit's own build, and
// it depends on nothing, so loading it works offline.
module mizu.test/api

go 1.27
