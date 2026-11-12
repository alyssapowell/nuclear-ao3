// Package export turns a work into a file a reader can keep: HTML, EPUB 3 or
// plain text. It only formats. Who may download a work, which chapters they
// get and whether a name is shown are decided by the caller (the works
// package) before a Work is built, so nothing here can leak what it is not
// handed.
//
// Everything that is not already HTML (titles, names, tags, labels) is
// escaped. Everything that is HTML (summary, notes, chapter text) is cleaned
// again with internal/html and then rewritten as well-formed XHTML, so a bad
// row in the database cannot put script or broken markup into a file.
package export
