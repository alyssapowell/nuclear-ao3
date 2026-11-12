package export

import (
	"archive/zip"
	"fmt"
	"hash/crc32"
	"io"
	"regexp"
	"strings"
	"time"
)

var validLang = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{1,8})*$`)

const (
	xmlDecl  = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	xhtmlNS  = `xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops"`
	epubMime = "application/epub+zip"
)

// EPUB writes the work as an EPUB 3 file: a zip with the mimetype entry
// first and stored, META-INF/container.xml, the package document, a
// navigation document, a stylesheet, a preface, one XHTML file per chapter
// and an afterword.
func EPUB(out io.Writer, w *Work) error {
	modified := w.Updated
	if modified.IsZero() {
		modified = time.Now()
	}
	modified = modified.UTC().Truncate(time.Second)

	zw := zip.NewWriter(out)

	// The mimetype entry: first, stored, no extra field, no data descriptor.
	// Only the legacy DOS time fields are set: Modified would add an
	// "extended timestamp" extra field, which the mimetype entry must not have.
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	if y := modified.Year(); y >= 1980 && y < 2100 {
		mh.ModifiedDate = uint16(y-1980)<<9 | uint16(modified.Month())<<5 | uint16(modified.Day())
		mh.ModifiedTime = uint16(modified.Hour())<<11 | uint16(modified.Minute())<<5 | uint16(modified.Second()/2)
	}
	mh.CRC32 = crc32.ChecksumIEEE([]byte(epubMime))
	mh.CompressedSize64 = uint64(len(epubMime))
	mh.UncompressedSize64 = uint64(len(epubMime))
	mw, err := zw.CreateRaw(mh)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(mw, epubMime); err != nil {
		return err
	}

	add := func(name, content string) error {
		fw, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: modified})
		if err != nil {
			return err
		}
		_, err = io.WriteString(fw, content)
		return err
	}

	lang := w.lang()
	page := func(title, body string) string {
		return xmlDecl + `<!DOCTYPE html>` + "\n" +
			fmt.Sprintf(`<html %s lang="%s" xml:lang="%s">`, xhtmlNS, esc(lang), esc(lang)) + "\n" +
			`<head><meta charset="utf-8"/><title>` + esc(title) + `</title>` +
			`<link rel="stylesheet" type="text/css" href="style.css"/></head>` + "\n<body>\n" + body + "</body>\n</html>\n"
	}

	type item struct{ id, href, title, props string }
	items := []item{{"preface", "preface.xhtml", "Preface", ""}}
	files := [][2]string{{"OEBPS/preface.xhtml", page(w.Title, w.preface())}}
	for i, c := range w.Chapters {
		heading := w.chapterHeading(c)
		if heading == "" {
			heading = w.Title
		}
		props := ""
		if hasImage(c.ContentHTML) || hasImage(c.SummaryHTML) || hasImage(c.NotesHTML) || hasImage(c.EndNotesHTML) {
			props = "remote-resources"
		}
		it := item{fmt.Sprintf("chapter%d", i+1), fmt.Sprintf("chapter%d.xhtml", i+1), heading, props}
		items = append(items, it)
		files = append(files, [2]string{"OEBPS/" + it.href, page(heading, w.chapter(c))})
	}
	if hasImage(w.SummaryHTML) || hasImage(w.NotesHTML) {
		items[0].props = "remote-resources"
	}
	items = append(items, item{"afterword", "afterword.xhtml", "Afterword", ""})
	files = append(files, [2]string{"OEBPS/afterword.xhtml", page("Afterword", w.afterword())})

	container := xmlDecl + `<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">` + "\n" +
		`  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>` + "\n</container>\n"
	if err := add("META-INF/container.xml", container); err != nil {
		return err
	}

	// Navigation document.
	var nav strings.Builder
	nav.WriteString(`<nav epub:type="toc" id="toc"><h1>Contents</h1><ol>` + "\n")
	for _, it := range items {
		fmt.Fprintf(&nav, `<li><a href="%s">%s</a></li>`+"\n", it.href, esc(it.title))
	}
	nav.WriteString("</ol></nav>\n")
	if err := add("OEBPS/nav.xhtml", page("Contents", nav.String())); err != nil {
		return err
	}
	if err := add("OEBPS/style.css", stylesheet); err != nil {
		return err
	}
	for _, f := range files {
		if err := add(f[0], f[1]); err != nil {
			return err
		}
	}

	// Package document.
	var opf strings.Builder
	opf.WriteString(xmlDecl)
	fmt.Fprintf(&opf, `<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="bookid" xml:lang="%s">`+"\n", esc(lang))
	opf.WriteString(`<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">` + "\n")
	fmt.Fprintf(&opf, `<dc:identifier id="bookid">urn:uuid:%s</dc:identifier>`+"\n", esc(w.ID))
	fmt.Fprintf(&opf, "<dc:title>%s</dc:title>\n", esc(w.Title))
	fmt.Fprintf(&opf, "<dc:language>%s</dc:language>\n", esc(lang))
	for _, a := range w.authors() {
		fmt.Fprintf(&opf, "<dc:creator>%s</dc:creator>\n", esc(a))
	}
	if !w.Published.IsZero() {
		fmt.Fprintf(&opf, "<dc:date>%s</dc:date>\n", w.Published.UTC().Format("2006-01-02"))
	}
	if s := plain(w.SummaryHTML); s != "" {
		fmt.Fprintf(&opf, "<dc:description>%s</dc:description>\n", esc(s))
	}
	for _, group := range [][]string{w.Fandoms, w.Relationships, w.Characters, w.Tags} {
		for _, t := range group {
			fmt.Fprintf(&opf, "<dc:subject>%s</dc:subject>\n", esc(t))
		}
	}
	fmt.Fprintf(&opf, `<meta property="dcterms:modified">%s</meta>`+"\n", modified.Format("2006-01-02T15:04:05Z"))
	opf.WriteString("</metadata>\n<manifest>\n")
	opf.WriteString(`<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>` + "\n")
	opf.WriteString(`<item id="css" href="style.css" media-type="text/css"/>` + "\n")
	for _, it := range items {
		props := ""
		if it.props != "" {
			props = fmt.Sprintf(` properties="%s"`, it.props)
		}
		fmt.Fprintf(&opf, `<item id="%s" href="%s" media-type="application/xhtml+xml"%s/>`+"\n", it.id, it.href, props)
	}
	opf.WriteString("</manifest>\n<spine>\n")
	for _, it := range items {
		fmt.Fprintf(&opf, `<itemref idref="%s"/>`+"\n", it.id)
	}
	opf.WriteString("</spine>\n</package>\n")
	if err := add("OEBPS/content.opf", opf.String()); err != nil {
		return err
	}
	return zw.Close()
}
