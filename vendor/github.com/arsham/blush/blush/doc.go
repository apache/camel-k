// Package blush reads from a given io.Reader line by line and looks for
// patterns.
//
// Blush struct has a Reader property which can be Stdin in case of it being
// shell's pipe, or any type that implements io.ReadCloser. If NoCut is set to
// true, it will show all lines despite being not matched. You cannot call
// Read() and WriteTo() on the same object. Blush will return ErrReadWriteMix on
// the second consequent call. The first time Read/WriteTo is called, it will
// start a goroutine and reads up to LineCache lines from Reader. If the Read()
// is in use, it starts a goroutine that reads up to CharCache bytes from the
// line cache and fills up the given buffer.
//
// The hex number should be in 3 or 6 part format (#aaaaaa or #aaa) and each
// part will be translated to a number value between 0 and 255 when creating the
// Colour instance. If any of hex parts are not between 00 and ff, it creates
// the DefaultColour value.
//
// Important Notes
//
// The Read() method could be slow in case of huge inspections. It is
// recommended to avoid it and use WriteTo() instead; io.Copy() can take care of
// that for you.
//
// When WriteTo() is called with an unavailable or un-writeable writer, there
// will be no further checks until it tries to write into it. If the Write
// encounters any errors regarding writes, it will return the amount if writes
// and stops its search.
//
// There always will be a newline after each read.
package blush
