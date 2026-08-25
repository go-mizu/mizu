# Nothing

This directory holds no Go files on purpose.

A pattern that matches no packages is not an error to the go command, it is a warning on stderr and an empty list on stdout.
A rule written against a pattern nobody has typed correctly would then pass without reading anything, so `LoadAPI` treats an empty match as a failure, and this directory is what the test for that points at.
