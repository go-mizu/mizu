package diag

// The allocation table. Adding a diagnostic means adding a line here, in the
// same commit, and the tests in registry_test.go say what a line has to look
// like.
//
// 1. Taking a code
//
// Find the block for the subsystem, take the next free number in it, and write
// the line. Nothing here is generated and nothing here is sorted for you: keep
// the list in code order and the tests will tell you when it is not.
//
// 2. Giving one up
//
// You do not. Set Retired to what replaced it or why it went away and leave the
// line where it is. A code that has been printed once is in somebody's notes,
// in a search index, and in the training data of whatever is reading their
// screen, and handing it to a different diagnostic makes all three wrong at
// once. The golden file in testdata holds the whole table, so a deleted line
// shows up in a diff as a deleted line rather than as a passing test.
//
// 3. Running out
//
// A block is a thousand codes and none of them is close. If one ever fills, the
// answer is MZ0xxx, which is held back for exactly this and is otherwise
// unallocated because a code with a leading zero reads like a placeholder. The
// answer is not renumbering, for the reason in section 2.
var subsystems = []Subsystem{
	{Name: "configuration", Doc: "05", Low: 1000, High: 1999},
	{Name: "the database, the query builder and the ORM", Doc: "11", Low: 2000, High: 2999},
	{Name: "HTTP routing, binding, validation and responses", Doc: "07", Low: 3000, High: 3999},
	{Name: "RPC and gRPC", Doc: "38", Low: 4000, High: 4999},
	{Name: "views, assets and frontend interoperability", Doc: "09", Low: 5000, High: 5999},
	{Name: "sessions, authentication, authorization and crypto", Doc: "10", Low: 6000, High: 6999},
	{Name: "caches, locks, queues, jobs, the scheduler and events", Doc: "15", Low: 7000, High: 7999},
	{Name: "mail, notifications, storage, telemetry, translations and flags", Doc: "19", Low: 8000, High: 8999},
	{Name: "the toolchain, the console and the generators", Doc: "23", Low: 9000, High: 9999},
}

// The codes.
//
// MZ1042 is out of order and is the oldest thing in this file. It was written
// into doc 36 and doc 37 as the worked example long before there was a registry
// to allocate it from, so by the time there was one it had been in print for
// months. Section 2 above applies to a code that has been printed once whether
// or not the thing that printed it was a design document, so it keeps the
// number it was published with and configuration starts its ordinary run at
// MZ1001. The gap between them is not going to be closed.
var entries = []Entry{
	{
		Code:    "MZ1001",
		Summary: "a configuration file is named but cannot be read",
	},
	{
		Code:    "MZ1002",
		Summary: "a configuration file is not valid TOML",
	},
	{
		Code:    "MZ1003",
		Summary: "an environment file has a line that is not a setting",
	},
	{
		Code:    "MZ1004",
		Summary: "a setting holds a value of the wrong kind for the field that reads it",
	},
	{
		Code:    "MZ1005",
		Summary: "a setting a program asked for is not set in any source",
	},
	{
		Code:    "MZ1006",
		Summary: "a command line flag was given with no value after it",
	},
	{
		Code:    "MZ1007",
		Summary: "a setting reads its value from a command and running commands is turned off",
	},
	{
		Code:    "MZ1042",
		Summary: "a setting is written down that nothing asked for",
	},
}
