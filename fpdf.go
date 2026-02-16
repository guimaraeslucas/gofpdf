/*
 * Gofpdf Standalone Library
 * Copyright (C) 2026 G3pix Ltda. All rights reserved.
 *
 * Developed by Lucas Guimarães - G3pix Ltda
 * Contact: https://g3pix.com.br
 * Project URL: https://github.com/guimaraeslucas/gofpdf
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 *
 * Third-Party Notice:
 * This is our translated version from PHP to Go of FPDF (v1.86),
 * originally authored by Olivier Plathey.
 */

// Package gofpdf provides a pure Go implementation for PDF document generation.
// It is a translation of the popular FPDF PHP library to Go with some addons.
// It includes support to HTML, images and tables
package gofpdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	stdhtml "html"
	"image"
	_ "image/gif"
	stdjpeg "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type pdfUVRange struct {
	start int
	count int
}

type pdfFont struct {
	typ       string
	name      string
	up        float64
	ut        float64
	cw        [256]int
	enc       string
	uv        map[int]interface{}
	subsetted bool
	n         int
	i         int
	file      string
	diff      string
}

type pdfImage struct {
	w    int
	h    int
	cs   string
	bpc  int
	f    string
	dp   string
	pal  []byte
	trns []int
	data []byte
	smk  []byte
	n    int
	i    int
}

// Fpdf is the main structure for PDF generation.
type Fpdf struct {
	state   int
	page    int
	n       int
	offsets map[int]int
	buffer  bytes.Buffer
	pages   map[int][]string

	compress bool
	k        float64

	defOrientation string
	curOrientation string
	stdPageSizes   map[string][2]float64
	defPageSize    [2]float64
	curPageSize    [2]float64
	curRotation    int
	pageInfo       map[int]map[string]interface{}

	wPt float64
	hPt float64
	w   float64
	h   float64

	lMargin float64
	tMargin float64
	rMargin float64
	bMargin float64
	cMargin float64

	x     float64
	y     float64
	lasth float64

	lineWidth float64
	fontpath  string

	coreFonts []string
	fonts     map[string]*pdfFont
	fontFiles map[string]map[string]int
	encodings map[string]int
	cmaps     map[string]int

	fontFamily  string
	fontStyle   string
	underline   bool
	currentFont *pdfFont
	fontSizePt  float64
	fontSize    float64

	drawColor string
	fillColor string
	textColor string
	colorFlag bool
	withAlpha bool
	ws        float64

	images map[string]*pdfImage

	pageLinks map[int][][]interface{}
	links     map[int][2]float64

	autoPageBreak    bool
	pageBreakTrigger float64
	inHeader         bool
	inFooter         bool
	aliasNbPages     string
	zoomMode         interface{}
	layoutMode       string
	metadata         map[string]string
	creationDate     time.Time
	pdfVersion       string

	assetFonts map[string]*pdfFont
	lastError  string

	// Hooks for Header and Footer
	headerFunc func()
	footerFunc func()
}

// NewFpdf creates a new PDF document.
// orientation: "P" for Portrait, "L" for Landscape.
// unit: "pt", "mm", "cm", "in".
// size: "A3", "A4", "A5", "Letter", "Legal".
func NewFpdf(orientation, unit, size string) *Fpdf {
	p := &Fpdf{}
	p.Reset(orientation, unit, size)
	return p
}

// Reset resets the PDF document with new parameters.
func (p *Fpdf) Reset(orientation, unit, size string) {
	p.state = 0
	p.page = 0
	p.n = 2
	p.offsets = map[int]int{}
	p.buffer.Reset()
	p.pages = map[int][]string{}
	p.pageInfo = map[int]map[string]interface{}{}
	p.fonts = map[string]*pdfFont{}
	p.fontFiles = map[string]map[string]int{}
	p.encodings = map[string]int{}
	p.cmaps = map[string]int{}
	p.images = map[string]*pdfImage{}
	p.links = map[int][2]float64{}
	p.pageLinks = map[int][][]interface{}{}
	p.inHeader = false
	p.inFooter = false
	p.lasth = 0
	p.fontFamily = ""
	p.fontStyle = ""
	p.fontSizePt = 12
	p.underline = false
	p.drawColor = "0 G"
	p.fillColor = "0 g"
	p.textColor = "0 g"
	p.colorFlag = false
	p.withAlpha = false
	p.ws = 0
	p.fontpath = ""
	p.coreFonts = []string{"courier", "helvetica", "times", "symbol", "zapfdingbats"}
	p.assetFonts = translatedFPDFFonts()

	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "pt":
		p.k = 1
	case "mm":
		p.k = 72.0 / 25.4
	case "cm":
		p.k = 72.0 / 2.54
	case "in":
		p.k = 72
	default:
		p.setError("incorrect unit: " + unit)
		p.k = 72.0 / 25.4
	}

	p.stdPageSizes = map[string][2]float64{
		"a3":     {841.89 / p.k, 1190.55 / p.k},
		"a4":     {595.28 / p.k, 841.89 / p.k},
		"a5":     {420.94 / p.k, 595.28 / p.k},
		"letter": {612.0 / p.k, 792.0 / p.k},
		"legal":  {612.0 / p.k, 1008.0 / p.k},
	}

	sz := p.getPageSize(size)
	p.defPageSize = sz
	p.curPageSize = sz

	o := strings.ToLower(strings.TrimSpace(orientation))
	if o == "" || o == "p" || o == "portrait" {
		p.defOrientation = "P"
		p.w = sz[0]
		p.h = sz[1]
	} else {
		p.defOrientation = "L"
		p.w = sz[1]
		p.h = sz[0]
	}
	p.curOrientation = p.defOrientation
	p.wPt = p.w * p.k
	p.hPt = p.h * p.k
	p.curRotation = 0

	margin := 28.35 / p.k
	p.SetMargins(margin, margin, nil)
	p.cMargin = margin / 10
	p.lineWidth = 0.567 / p.k
	p.SetAutoPageBreak(true, 2*margin)
	p.SetDisplayMode("default", "default")
	p.SetCompression(true)
	p.metadata = map[string]string{"Producer": "G3pix Gofpdf Library"}
	p.pdfVersion = "1.3"
	p.creationDate = time.Now()
	p.lastError = ""
}

// SetHeaderFunc sets a custom header function.
func (p *Fpdf) SetHeaderFunc(f func()) { p.headerFunc = f }

// SetFooterFunc sets a custom footer function.
func (p *Fpdf) SetFooterFunc(f func()) { p.footerFunc = f }

// GetX returns the current X position.
func (p *Fpdf) GetX() float64 { return p.x }

// GetY returns the current Y position.
func (p *Fpdf) GetY() float64 { return p.y }

// SetX sets the X position.
func (p *Fpdf) SetX(x float64) {
	if x >= 0 {
		p.x = x
	} else {
		p.x = p.w + x
	}
}

// SetY sets the Y position. If resetX is true, X is reset to the left margin.
func (p *Fpdf) SetY(y float64, resetX bool) {
	if y >= 0 {
		p.y = y
	} else {
		p.y = p.h + y
	}
	if resetX {
		p.x = p.lMargin
	}
}

// SetXY sets both X and Y positions.
func (p *Fpdf) SetXY(x, y float64) {
	p.SetX(x)
	p.SetY(y, false)
}

// AddPage adds a new page to the document.
func (p *Fpdf) AddPage(orientation, size string, rotation int) {
	if p.state == 3 {
		p.panicError("the document is closed")
	}
	family := p.fontFamily
	style := p.fontStyle
	if p.underline {
		style += "U"
	}
	fontsize := p.fontSizePt
	lw := p.lineWidth
	dc := p.drawColor
	fc := p.fillColor
	tc := p.textColor
	cf := p.colorFlag
	if p.page > 0 {
		p.inFooter = true
		p.Footer()
		p.inFooter = false
		p.endPage()
	}
	p.beginPage(orientation, size, rotation)
	p.out("2 J")
	p.lineWidth = lw
	p.out(sprintf("%.2F w", lw*p.k))
	if family != "" {
		p.SetFont(family, style, fontsize)
	}
	p.drawColor = dc
	if dc != "0 G" {
		p.out(dc)
	}
	p.fillColor = fc
	if fc != "0 g" {
		p.out(fc)
	}
	p.textColor = tc
	p.colorFlag = cf

	p.inHeader = true
	p.Header()
	p.inHeader = false

	if p.lineWidth != lw {
		p.lineWidth = lw
		p.out(sprintf("%.2F w", lw*p.k))
	}
	if family != "" {
		p.SetFont(family, style, fontsize)
	}
	if p.drawColor != dc {
		p.drawColor = dc
		p.out(dc)
	}
	if p.fillColor != fc {
		p.fillColor = fc
		p.out(fc)
	}
	p.textColor = tc
	p.colorFlag = cf
}

// Header is called automatically when a new page is added.
func (p *Fpdf) Header() {
	if p.headerFunc != nil {
		p.headerFunc()
	}
}

// Footer is called automatically before a page break or closing the document.
func (p *Fpdf) Footer() {
	if p.footerFunc != nil {
		p.footerFunc()
	}
}

// SetMargins sets the left, top and optionally right margins.
func (p *Fpdf) SetMargins(left, top float64, right *float64) {
	p.lMargin = left
	p.tMargin = top
	if right == nil {
		p.rMargin = left
	} else {
		p.rMargin = *right
	}
}

// SetAutoPageBreak sets the auto page break mode and the bottom margin.
func (p *Fpdf) SetAutoPageBreak(auto bool, margin float64) {
	p.autoPageBreak = auto
	p.bMargin = margin
	p.pageBreakTrigger = p.h - margin
}

// SetFont sets the font family, style and size.
func (p *Fpdf) SetFont(family, style string, size float64) {
	if family == "" {
		family = p.fontFamily
	} else {
		family = strings.ToLower(family)
	}
	style = strings.ToUpper(style)
	if strings.Contains(style, "U") {
		p.underline = true
		style = strings.ReplaceAll(style, "U", "")
	} else {
		p.underline = false
	}
	if style == "IB" {
		style = "BI"
	}
	if size == 0 {
		size = p.fontSizePt
	}
	if p.fontFamily == family && p.fontStyle == style && p.fontSizePt == size {
		return
	}
	fontkey := family + style
	if _, ok := p.fonts[fontkey]; !ok {
		if family == "arial" {
			family = "helvetica"
		}
		if containsString(p.coreFonts, family) {
			if family == "symbol" || family == "zapfdingbats" {
				style = ""
			}
			fontkey = family + style
			if _, ok2 := p.fonts[fontkey]; !ok2 {
				p.AddFont(family, style, "", "")
			}
		} else {
			p.panicError("undefined font: " + family + " " + style)
		}
	}
	p.fontFamily = family
	p.fontStyle = style
	p.fontSizePt = size
	p.fontSize = size / p.k
	p.currentFont = p.fonts[fontkey]
	if p.page > 0 {
		p.out(sprintf("BT /F%d %.2F Tf ET", p.currentFont.i, p.fontSizePt))
	}
}

// SetFontSize sets the font size.
func (p *Fpdf) SetFontSize(size float64) {
	if p.fontSizePt == size {
		return
	}
	p.fontSizePt = size
	p.fontSize = size / p.k
	if p.page > 0 && p.currentFont != nil {
		p.out(sprintf("BT /F%d %.2F Tf ET", p.currentFont.i, p.fontSizePt))
	}
}

// SetTextColor sets the text color (RGB).
func (p *Fpdf) SetTextColor(r, g, b float64) {
	if math.IsNaN(g) || (r == 0 && g == 0 && b == 0) {
		p.textColor = sprintf("%.3F g", r/255)
	} else {
		p.textColor = sprintf("%.3F %.3F %.3F rg", r/255, g/255, b/255)
	}
	p.colorFlag = p.fillColor != p.textColor
}

// SetFillColor sets the fill color (RGB).
func (p *Fpdf) SetFillColor(r, g, b float64) {
	if math.IsNaN(g) || (r == 0 && g == 0 && b == 0) {
		p.fillColor = sprintf("%.3F g", r/255)
	} else {
		p.fillColor = sprintf("%.3F %.3F %.3F rg", r/255, g/255, b/255)
	}
	p.colorFlag = p.fillColor != p.textColor
	if p.page > 0 {
		p.out(p.fillColor)
	}
}

// SetDrawColor sets the draw color (RGB).
func (p *Fpdf) SetDrawColor(r, g, b float64) {
	if math.IsNaN(g) || (r == 0 && g == 0 && b == 0) {
		p.drawColor = sprintf("%.3F G", r/255)
	} else {
		p.drawColor = sprintf("%.3F %.3F %.3F RG", r/255, g/255, b/255)
	}
	if p.page > 0 {
		p.out(p.drawColor)
	}
}

// SetLineWidth sets the line width.
func (p *Fpdf) SetLineWidth(width float64) {
	p.lineWidth = width
	if p.page > 0 {
		p.out(sprintf("%.2F w", width*p.k))
	}
}

// Line draws a line.
func (p *Fpdf) Line(x1, y1, x2, y2 float64) {
	p.out(sprintf("%.2F %.2F m %.2F %.2F l S", x1*p.k, (p.h-y1)*p.k, x2*p.k, (p.h-y2)*p.k))
}

// Rect draws a rectangle. style: "D" or empty for draw, "F" for fill, "DF" or "FD" for both.
func (p *Fpdf) Rect(x, y, w, h float64, style string) {
	op := "S"
	switch style {
	case "F":
		op = "f"
	case "FD", "DF":
		op = "B"
	}
	p.out(sprintf("%.2F %.2F %.2F %.2F re %s", x*p.k, (p.h-y)*p.k, w*p.k, -h*p.k, op))
}

// Text prints a string at a specific position.
func (p *Fpdf) Text(x, y float64, txt string) {
	if p.currentFont == nil {
		p.panicError("no font has been set")
	}
	s := sprintf("BT %.2F %.2F Td (%s) Tj ET", x*p.k, (p.h-y)*p.k, p.escape(txt))
	if p.underline && txt != "" {
		s += " " + p.doUnderline(x, y, txt)
	}
	if p.colorFlag {
		s = "q " + p.textColor + " " + s + " Q"
	}
	p.out(s)
}

// Cell prints a cell (rectangular area) with optional borders and background.
func (p *Fpdf) Cell(w, h float64, txt string, border interface{}, ln int, align string, fill bool, link interface{}) {
	k := p.k
	if p.y+h > p.pageBreakTrigger && !p.inHeader && !p.inFooter && p.AcceptPageBreak() {
		x := p.x
		ws := p.ws
		if ws > 0 {
			p.ws = 0
			p.out("0 Tw")
		}
		p.AddPage(p.curOrientation, "", p.curRotation)
		p.x = x
		if ws > 0 {
			p.ws = ws
			p.out(sprintf("%.3F Tw", ws*k))
		}
	}
	if w == 0 {
		w = p.w - p.rMargin - p.x
	}
	s := ""
	if fill || border == 1 || border == "1" {
		op := "S"
		if fill {
			if border == 1 || border == "1" {
				op = "B"
			} else {
				op = "f"
			}
		}
		s = sprintf("%.2F %.2F %.2F %.2F re %s ", p.x*k, (p.h-p.y)*k, w*k, -h*k, op)
	}
	if bs, ok := border.(string); ok {
		x := p.x
		y := p.y
		if strings.Contains(bs, "L") {
			s += sprintf("%.2F %.2F m %.2F %.2F l S ", x*k, (p.h-y)*k, x*k, (p.h-(y+h))*k)
		}
		if strings.Contains(bs, "T") {
			s += sprintf("%.2F %.2F m %.2F %.2F l S ", x*k, (p.h-y)*k, (x+w)*k, (p.h-y)*k)
		}
		if strings.Contains(bs, "R") {
			s += sprintf("%.2F %.2F m %.2F %.2F l S ", (x+w)*k, (p.h-y)*k, (x+w)*k, (p.h-(y+h))*k)
		}
		if strings.Contains(bs, "B") {
			s += sprintf("%.2F %.2F m %.2F %.2F l S ", x*k, (p.h-(y+h))*k, (x+w)*k, (p.h-(y+h))*k)
		}
	}
	if txt != "" {
		if p.currentFont == nil {
			p.panicError("no font has been set")
		}
		dx := p.cMargin
		switch align {
		case "R":
			dx = w - p.cMargin - p.GetStringWidth(txt)
		case "C":
			dx = (w - p.GetStringWidth(txt)) / 2
		}
		if p.colorFlag {
			s += "q " + p.textColor + " "
		}
		s += sprintf("BT %.2F %.2F Td (%s) Tj ET", (p.x+dx)*k, (p.h-(p.y+0.5*h+0.3*p.fontSize))*k, p.escape(txt))
		if p.underline {
			s += " " + p.doUnderline(p.x+dx, p.y+0.5*h+0.3*p.fontSize, txt)
		}
		if p.colorFlag {
			s += " Q"
		}
		if link != "" && link != nil {
			p.Link(p.x+dx, p.y+0.5*h-0.5*p.fontSize, p.GetStringWidth(txt), p.fontSize, link)
		}
	}
	if s != "" {
		p.out(s)
	}
	p.lasth = h
	if ln > 0 {
		p.y += h
		if ln == 1 {
			p.x = p.lMargin
		}
	} else {
		p.x += w
	}
}

// MultiCell prints text with line breaks.
func (p *Fpdf) MultiCell(w, h float64, txt string, border interface{}, align string, fill bool) {
	if p.currentFont == nil {
		p.panicError("no font has been set")
	}
	if w == 0 {
		w = p.w - p.rMargin - p.x
	}
	wmax := (w - 2*p.cMargin) * 1000 / p.fontSize
	s := strings.ReplaceAll(txt, "\r", "")
	nb := len(s)
	if nb > 0 && s[nb-1] == '\n' {
		nb--
	}
	b := ""
	b2 := ""
	if border != nil && border != 0 && border != "0" && border != "" {
		if border == 1 || border == "1" {
			b = "LRT"
			b2 = "LR"
		} else if bs, ok := border.(string); ok {
			if strings.Contains(bs, "L") {
				b2 += "L"
			}
			if strings.Contains(bs, "R") {
				b2 += "R"
			}
			if strings.Contains(bs, "T") {
				b = b2 + "T"
			} else {
				b = b2
			}
		}
	}
	sep := -1
	i, j := 0, 0
	l, ns, nl := 0, 0, 1
	for i < nb {
		c := s[i]
		if c == '\n' {
			if p.ws > 0 {
				p.ws = 0
				p.out("0 Tw")
			}
			p.Cell(w, h, s[j:i], b, 2, align, fill, "")
			i++
			sep = -1
			j = i
			l = 0
			ns = 0
			nl++
			if b != "" && nl == 2 {
				b = b2
			}
			continue
		}
		if c == ' ' {
			sep = i
			ns++
		}
		l += p.charWidth(c)
		if float64(l) > wmax {
			if sep == -1 {
				if i == j {
					i++
				}
				if p.ws > 0 {
					p.ws = 0
					p.out("0 Tw")
				}
				p.Cell(w, h, s[j:i], b, 2, align, fill, "")
			} else {
				if align == "J" {
					spaces := strings.Count(s[j:sep], " ")
					if spaces > 0 {
						strW := p.GetStringWidth(s[j:sep])
						p.ws = (w - 2*p.cMargin - strW) / float64(spaces)
						p.out(sprintf("%.3F Tw", p.ws*p.k))
					}
				}
				p.Cell(w, h, s[j:sep], b, 2, align, fill, "")
				i = sep + 1
			}
			sep = -1
			j = i
			l = 0
			ns = 0
			nl++
			if b != "" && nl == 2 {
				b = b2
			}
		} else {
			i++
		}
	}
	if p.ws > 0 {
		p.ws = 0
		p.out("0 Tw")
	}
	if border == 1 || border == "1" {
		b += "B"
	} else if bs, ok := border.(string); ok && strings.Contains(bs, "B") {
		b += "B"
	}
	p.Cell(w, h, s[j:i], b, 2, align, fill, "")
	p.x = p.lMargin
}

// Write prints text from the current position.
func (p *Fpdf) Write(h float64, txt string, link interface{}) {
	if p.currentFont == nil {
		p.panicError("no font has been set")
	}
	w := p.w - p.rMargin - p.x
	wmax := (w - 2*p.cMargin) * 1000 / p.fontSize
	s := strings.ReplaceAll(txt, "\r", "")
	nb := len(s)
	sep := -1
	i, j, l, nl := 0, 0, 0, 1
	for i < nb {
		c := s[i]
		if c == '\n' {
			p.Cell(w, h, s[j:i], 0, 2, "", false, link)
			i++
			sep = -1
			j = i
			l = 0
			if nl == 1 {
				p.x = p.lMargin
				w = p.w - p.rMargin - p.x
				wmax = (w - 2*p.cMargin) * 1000 / p.fontSize
			}
			nl++
			continue
		}
		if c == ' ' {
			sep = i
		}
		l += p.charWidth(c)
		if float64(l) > wmax {
			if sep == -1 {
				if p.x > p.lMargin {
					p.x = p.lMargin
					p.y += h
					w = p.w - p.rMargin - p.x
					wmax = (w - 2*p.cMargin) * 1000 / p.fontSize
					i++
					nl++
					continue
				}
				if i == j {
					i++
				}
				p.Cell(w, h, s[j:i], 0, 2, "", false, link)
			} else {
				p.Cell(w, h, s[j:sep], 0, 2, "", false, link)
				i = sep + 1
			}
			sep = -1
			j = i
			l = 0
			if nl == 1 {
				p.x = p.lMargin
				w = p.w - p.rMargin - p.x
				wmax = (w - 2*p.cMargin) * 1000 / p.fontSize
			}
			nl++
		} else {
			i++
		}
	}
	if i != j {
		p.Cell(float64(l)/1000*p.fontSize, h, s[j:], 0, 0, "", false, link)
	}
}

// Image inserts an image into the document.
func (p *Fpdf) Image(file string, x, y, w, h float64, typ string, link interface{}) {
	if file == "" {
		p.panicError("image file name is empty")
	}
	info, ok := p.images[file]
	if !ok {
		if typ == "" {
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
			if ext == "" {
				p.panicError("image file has no extension and no type was specified: " + file)
			}
			typ = ext
		}
		typ = strings.ToLower(typ)
		if typ == "jpeg" {
			typ = "jpg"
		}
		switch typ {
		case "jpg", "png", "gif":
			info = p.parseImageFile(file)
		default:
			p.panicError("unsupported image type: " + typ)
		}
		info.i = len(p.images) + 1
		p.images[file] = info
	}

	if w == 0 && h == 0 {
		w = -96
		h = -96
	}
	if w < 0 {
		w = -float64(info.w) * 72 / w / p.k
	}
	if h < 0 {
		h = -float64(info.h) * 72 / h / p.k
	}
	if w == 0 {
		w = h * float64(info.w) / float64(info.h)
	}
	if h == 0 {
		h = w * float64(info.h) / float64(info.w)
	}
	if math.IsNaN(y) {
		if p.y+h > p.pageBreakTrigger && !p.inHeader && !p.inFooter && p.AcceptPageBreak() {
			x2 := p.x
			p.AddPage(p.curOrientation, "", p.curRotation)
			p.x = x2
		}
		y = p.y
		p.y += h
	}
	if math.IsNaN(x) {
		x = p.x
	}
	p.out(sprintf("q %.2F 0 0 %.2F %.2F %.2F cm /I%d Do Q", w*p.k, h*p.k, x*p.k, (p.h-(y+h))*p.k, info.i))
	if link != "" && link != nil {
		p.Link(x, y, w, h, link)
	}
}

// Ln performs a line break.
func (p *Fpdf) Ln(h float64) {
	p.x = p.lMargin
	if h < 0 {
		p.y += p.lasth
	} else {
		p.y += h
	}
}

// GetStringWidth returns the width of a string in the current font.
func (p *Fpdf) GetStringWidth(s string) float64 {
	if p.currentFont == nil {
		return 0
	}
	w := 0
	for _, c := range []byte(s) {
		w += p.currentFont.cw[c]
	}
	return float64(w) * p.fontSize / 1000
}

// AddFont adds a font to the document.
func (p *Fpdf) AddFont(family, style, file, dir string) {
	family = strings.ToLower(strings.TrimSpace(family))
	if file == "" {
		file = strings.ReplaceAll(family, " ", "") + strings.ToLower(style) + ".php"
	}
	style = strings.ToUpper(style)
	if style == "IB" {
		style = "BI"
	}
	fontkey := family + style
	if _, ok := p.fonts[fontkey]; ok {
		return
	}
	if strings.Contains(file, "/") || strings.Contains(file, "\\") {
		p.panicError("incorrect font definition file name: " + file)
	}
	if dir == "" {
		dir = p.fontpath
	}
	info, ok := p.loadFontAsset(file)
	if !ok {
		p.panicError("could not load embedded font definition: " + file)
	}
	clone := *info
	clone.i = len(p.fonts) + 1
	p.fonts[fontkey] = &clone
}

// Close closes the document.
func (p *Fpdf) Close() {
	if p.state == 3 {
		return
	}
	if p.page == 0 {
		p.AddPage("", "", 0)
	}
	p.inFooter = true
	p.Footer()
	p.inFooter = false
	p.endPage()
	p.endDoc()
}

// Output exports the PDF document. dest can be "S" (string), "F" (file), or empty (default "S").
func (p *Fpdf) Output(dest, name string) (string, error) {
	p.Close()
	if dest == "" {
		dest = "S"
	}
	pdf := p.buffer.Bytes()
	switch strings.ToUpper(dest) {
	case "F":
		if err := os.WriteFile(name, pdf, 0644); err != nil {
			return "", err
		}
		return "", nil
	case "S":
		return string(pdf), nil
	default:
		return "", fmt.Errorf("incorrect output destination: %s", dest)
	}
}

// AcceptPageBreak is called automatically when a page break is needed.
func (p *Fpdf) AcceptPageBreak() bool { return p.autoPageBreak }

// Link adds a clickable link to the document.
func (p *Fpdf) Link(x, y, w, h float64, link interface{}) {
	p.pageLinks[p.page] = append(p.pageLinks[p.page], []interface{}{x * p.k, p.hPt - y*p.k, w * p.k, h * p.k, link})
}

// SetCompression sets whether to compress PDF page streams.
func (p *Fpdf) SetCompression(compress bool) { p.compress = compress }

// SetTitle sets the document title.
func (p *Fpdf) SetTitle(title string) { p.metadata["Title"] = p.metaText(title, false) }

// SetAuthor sets the document author.
func (p *Fpdf) SetAuthor(v string) { p.metadata["Author"] = p.metaText(v, false) }

// SetSubject sets the document subject.
func (p *Fpdf) SetSubject(v string) { p.metadata["Subject"] = p.metaText(v, false) }

// SetKeywords sets the document keywords.
func (p *Fpdf) SetKeywords(v string) { p.metadata["Keywords"] = p.metaText(v, false) }

// SetCreator sets the document creator.
func (p *Fpdf) SetCreator(v string) { p.metadata["Creator"] = p.metaText(v, false) }

// SetDisplayMode sets the display mode of the PDF viewer.
func (p *Fpdf) SetDisplayMode(zoom interface{}, layout string) {
	p.zoomMode = zoom
	p.layoutMode = strings.ToLower(layout)
}

// WriteHTML renders basic HTML into the PDF.
func (p *Fpdf) WriteHTML(htmlInput string) {
	if strings.TrimSpace(htmlInput) == "" {
		return
	}
	if p.page == 0 {
		p.AddPage("", "", 0)
	}

	state := &pdfHTMLState{
		p:               p,
		tdAlign:         "L",
		currAlign:       "L",
		defaultFontSize: p.fontSizePt,
		tableColWidths:  make(map[int]float64),
	}

	normalized := strings.ReplaceAll(htmlInput, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.ReplaceAll(normalized, "\t", "")

	state.renderHTML(normalized)
}

// Internal helpers follow (simplified for brevity)

func (p *Fpdf) getPageSize(size string) [2]float64 {
	s := strings.ToLower(strings.TrimSpace(size))
	if s == "" {
		s = "a4"
	}
	if v, ok := p.stdPageSizes[s]; ok {
		return v
	}
	return p.stdPageSizes["a4"]
}

func (p *Fpdf) beginPage(orientation, size string, rotation int) {
	p.page++
	p.pages[p.page] = []string{}
	p.pageLinks[p.page] = [][]any{}
	p.state = 2
	p.x = p.lMargin
	p.y = p.tMargin
	p.fontFamily = ""

	if orientation == "" {
		orientation = p.defOrientation
	} else {
		orientation = strings.ToUpper(string(orientation[0]))
	}

	var ps [2]float64
	if size == "" {
		ps = p.defPageSize
	} else {
		ps = p.getPageSize(size)
	}
	if orientation != p.curOrientation || ps != p.curPageSize {
		if orientation == "P" {
			p.w, p.h = ps[0], ps[1]
		} else {
			p.w, p.h = ps[1], ps[0]
		}
		p.wPt = p.w * p.k
		p.hPt = p.h * p.k
		p.pageBreakTrigger = p.h - p.bMargin
		p.curOrientation = orientation
		p.curPageSize = ps
	}
	if orientation != p.defOrientation || ps != p.defPageSize {
		if p.pageInfo[p.page] == nil {
			p.pageInfo[p.page] = map[string]interface{}{}
		}
		p.pageInfo[p.page]["size"] = [2]float64{p.wPt, p.hPt}
	}
	if rotation != 0 {
		if p.pageInfo[p.page] == nil {
			p.pageInfo[p.page] = map[string]interface{}{}
		}
		p.pageInfo[p.page]["rotation"] = rotation
	}
	p.curRotation = rotation
}

func (p *Fpdf) endPage() { p.state = 1 }

func (p *Fpdf) out(s string) {
	switch p.state {
	case 2:
		p.pages[p.page] = append(p.pages[p.page], s)
	case 0:
		p.panicError("no page has been added yet")
	case 1:
		p.panicError("invalid call")
	case 3:
		p.panicError("the document is closed")
	}
}

func (p *Fpdf) endDoc() {
	p.creationDate = time.Now()
	p.putHeader()
	p.putPages()
	p.putResources()
	p.newObj()
	p.put("<<")
	p.putInfo()
	p.put(">>")
	p.put("endobj")
	p.newObj()
	p.put("<<")
	p.putCatalog()
	p.put(">>")
	p.put("endobj")
	offset := p.getOffset()
	p.put("xref")
	p.put("0 " + strconv.Itoa(p.n+1))
	p.put("0000000000 65535 f ")
	for i := 1; i <= p.n; i++ {
		p.put(sprintf("%010d 00000 n ", p.offsets[i]))
	}
	p.put("trailer")
	p.put("<<")
	p.putTrailer()
	p.put(">>")
	p.put("startxref")
	p.put(strconv.Itoa(offset))
	p.put("%%EOF")
	p.state = 3
}

func (p *Fpdf) putHeader() { p.put("%PDF-" + p.pdfVersion) }
func (p *Fpdf) putTrailer() {
	p.put("/Size " + strconv.Itoa(p.n+1))
	p.put("/Root " + strconv.Itoa(p.n) + " 0 R")
	p.put("/Info " + strconv.Itoa(p.n-1) + " 0 R")
}
func (p *Fpdf) put(s string) {
	p.buffer.WriteString(s)
	p.buffer.WriteByte('\n')
}
func (p *Fpdf) getOffset() int { return p.buffer.Len() }
func (p *Fpdf) newObj(forced ...int) {
	n := 0
	if len(forced) > 0 {
		n = forced[0]
		p.n = maxInt(p.n, n)
	} else {
		p.n++
		n = p.n
	}
	p.offsets[n] = p.getOffset()
	p.put(strconv.Itoa(n) + " 0 obj")
}
func (p *Fpdf) putStream(data []byte) {
	p.put("stream")
	p.buffer.Write(data)
	p.buffer.WriteByte('\n')
	p.put("endstream")
}
func (p *Fpdf) putStreamObject(data []byte) {
	entries := ""
	if p.compress {
		entries = "/Filter /FlateDecode "
		data = flateCompress(data)
	}
	entries += "/Length " + strconv.Itoa(len(data))
	p.newObj()
	p.put("<<" + entries + ">>")
	p.putStream(data)
	p.put("endobj")
}

func (p *Fpdf) putPages() {
	n := p.n
	for i := 1; i <= p.page; i++ {
		if p.pageInfo[i] == nil {
			p.pageInfo[i] = map[string]interface{}{}
		}
		n++
		p.pageInfo[i]["n"] = n
		n++
		for idx := range p.pageLinks[i] {
			n++
			p.pageLinks[i][idx] = append(p.pageLinks[i][idx], n)
		}
	}
	for i := 1; i <= p.page; i++ {
		p.putPage(i)
	}
	p.newObj(1)
	p.put("<</Type /Pages")
	kids := "/Kids ["
	for i := 1; i <= p.page; i++ {
		kids += strconv.Itoa(toInt(p.pageInfo[i]["n"])) + " 0 R "
	}
	kids += "]"
	p.put(kids)
	p.put("/Count " + strconv.Itoa(p.page))
	w, h := p.defPageSize[0], p.defPageSize[1]
	if p.defOrientation != "P" {
		w, h = h, w
	}
	p.put(sprintf("/MediaBox [0 0 %.2F %.2F]", w*p.k, h*p.k))
	p.put(">>")
	p.put("endobj")
}

func (p *Fpdf) putPage(n int) {
	p.newObj()
	p.put("<</Type /Page")
	p.put("/Parent 1 0 R")
	if pi, ok := p.pageInfo[n]; ok {
		if sz, ok2 := pi["size"].([2]float64); ok2 {
			p.put(sprintf("/MediaBox [0 0 %.2F %.2F]", sz[0], sz[1]))
		}
		if rot, ok2 := pi["rotation"].(int); ok2 {
			p.put("/Rotate " + strconv.Itoa(rot))
		}
	}
	p.put("/Resources 2 0 R")
	if len(p.pageLinks[n]) > 0 {
		s := "/Annots ["
		for _, pl := range p.pageLinks[n] {
			s += strconv.Itoa(toInt(pl[5])) + " 0 R "
		}
		s += "]"
		p.put(s)
	}
	if p.withAlpha {
		p.put("/Group <</Type /Group /S /Transparency /CS /DeviceRGB>>")
	}
	p.put("/Contents " + strconv.Itoa(p.n+1) + " 0 R>>")
	p.put("endobj")

	content := strings.Join(p.pages[n], "\n") + "\n"
	if p.aliasNbPages != "" {
		content = strings.ReplaceAll(content, p.aliasNbPages, strconv.Itoa(p.page))
	}
	p.putStreamObject([]byte(content))
	p.putLinks(n)
}

func (p *Fpdf) putLinks(n int) {
	for _, pl := range p.pageLinks[n] {
		p.newObj()
		x := toFloat(pl[0])
		y := toFloat(pl[1])
		w := toFloat(pl[2])
		h := toFloat(pl[3])
		rect := sprintf("%.2F %.2F %.2F %.2F", x, y, x+w, y-h)
		s := "<</Type /Annot /Subtype /Link /Rect [" + rect + "] /Border [0 0 0] "
		switch v := pl[4].(type) {
		case string:
			s += "/A <</S /URI /URI " + p.textString(v) + ">>>>"
		default:
			lnk := toInt(v)
			dst := p.links[lnk]
			page := int(dst[0])
			y2 := dst[1]
			hPage := p.hPt
			if pi, ok := p.pageInfo[page]; ok {
				if sz, ok2 := pi["size"].([2]float64); ok2 {
					hPage = sz[1]
				}
			}
			nobj := p.pageInfo[page]["n"]
			s += sprintf("/Dest [%d 0 R /XYZ 0 %.2F null]>>", toInt(nobj), hPage-y2*p.k)
		}
		p.put(s)
		p.put("endobj")
	}
}

func (p *Fpdf) putResources() {
	p.putFonts()
	p.putImages()
	p.newObj(2)
	p.put("<<")
	p.putResourceDict()
	p.put(">>")
	p.put("endobj")
}

func (p *Fpdf) putFonts() {
	for k, f := range p.fonts {
		toUnicodeObj := 0
		if len(f.uv) > 0 {
			cmap := p.toUnicodeCMap(f.uv)
			p.putStreamObject([]byte(cmap))
			toUnicodeObj = p.n
		}

		p.newObj()
		f.n = p.n
		p.fonts[k] = f

		p.put("<</Type /Font")
		p.put("/BaseFont /" + f.name)
		p.put("/Subtype /Type1")
		if f.name != "Symbol" && f.name != "ZapfDingbats" {
			p.put("/Encoding /WinAnsiEncoding")
		}
		if toUnicodeObj > 0 {
			p.put("/ToUnicode " + strconv.Itoa(toUnicodeObj) + " 0 R")
		}
		p.put(">>")
		p.put("endobj")
	}
}

func (p *Fpdf) toUnicodeCMap(uv map[int]interface{}) string {
	var ranges strings.Builder
	var chars strings.Builder
	nbr, nbc := 0, 0
	keys := make([]int, 0, len(uv))
	for k := range uv {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, c := range keys {
		v := uv[c]
		switch vv := v.(type) {
		case pdfUVRange:
			ranges.WriteString(sprintf("<%02X> <%02X> <%04X>\n", c, c+vv.count-1, vv.start))
			nbr++
		case int:
			chars.WriteString(sprintf("<%02X> <%04X>\n", c, vv))
			nbc++
		}
	}
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\n")
	b.WriteString("begincmap\n")
	b.WriteString("/CIDSystemInfo\n<</Registry (Adobe)\n/Ordering (UCS)\n/Supplement 0\n>> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n<00> <FF>\nendcodespacerange\n")
	if nbr > 0 {
		b.WriteString(strconv.Itoa(nbr) + " beginbfrange\n")
		b.WriteString(ranges.String())
		b.WriteString("endbfrange\n")
	}
	if nbc > 0 {
		b.WriteString(strconv.Itoa(nbc) + " beginbfchar\n")
		b.WriteString(chars.String())
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend")
	return b.String()
}

func (p *Fpdf) putImages() {
	for _, info := range p.images {
		p.putImage(info)
	}
}

func (p *Fpdf) putImage(info *pdfImage) {
	p.newObj()
	info.n = p.n
	p.put("<</Type /XObject")
	p.put("/Subtype /Image")
	p.put("/Width " + strconv.Itoa(info.w))
	p.put("/Height " + strconv.Itoa(info.h))
	p.put("/ColorSpace /" + info.cs)
	p.put("/BitsPerComponent " + strconv.Itoa(info.bpc))
	if info.f != "" {
		p.put("/Filter /" + info.f)
	}
	p.put("/Length " + strconv.Itoa(len(info.data)) + ">>")
	p.putStream(info.data)
	p.put("endobj")
}

func (p *Fpdf) putResourceDict() {
	p.put("/ProcSet [/PDF /Text /ImageB /ImageC /ImageI]")
	p.put("/Font <<")
	for _, f := range p.fonts {
		p.put("/F" + strconv.Itoa(f.i) + " " + strconv.Itoa(f.n) + " 0 R")
	}
	p.put(">>")
	p.put("/XObject <<")
	for _, image := range p.images {
		p.put("/I" + strconv.Itoa(image.i) + " " + strconv.Itoa(image.n) + " 0 R")
	}
	p.put(">>")
}

func (p *Fpdf) putInfo() {
	date := p.creationDate.Format("20060102150405-0700")
	p.metadata["CreationDate"] = "D:" + date[:len(date)-2] + "'" + date[len(date)-2:] + "'"
	keys := make([]string, 0, len(p.metadata))
	for k := range p.metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p.put("/" + k + " " + p.textString(p.metadata[k]))
	}
}

func (p *Fpdf) putCatalog() {
	n := toInt(p.pageInfo[1]["n"])
	p.put("/Type /Catalog")
	p.put("/Pages 1 0 R")
	switch v := p.zoomMode.(type) {
	case string:
		s := strings.ToLower(v)
		switch s {
		case "fullpage":
			p.put("/OpenAction [" + strconv.Itoa(n) + " 0 R /Fit]")
		case "fullwidth":
			p.put("/OpenAction [" + strconv.Itoa(n) + " 0 R /FitH null]")
		case "real":
			p.put("/OpenAction [" + strconv.Itoa(n) + " 0 R /XYZ null null 1]")
		}
	case float64:
		p.put(sprintf("/OpenAction [%d 0 R /XYZ null null %.2F]", n, v/100))
	}
	switch p.layoutMode {
	case "single":
		p.put("/PageLayout /SinglePage")
	case "continuous":
		p.put("/PageLayout /OneColumn")
	case "two":
		p.put("/PageLayout /TwoColumnLeft")
	}
}

func (p *Fpdf) setError(msg string)   { p.lastError = msg }
func (p *Fpdf) panicError(msg string) { panic("fpdf error: " + msg) }

func (p *Fpdf) metaText(v string, isUTF8 bool) string {
	if isUTF8 {
		return v
	}
	return latin1ToUTF8(v)
}

func (p *Fpdf) escape(s string) string {
	r := strings.ReplaceAll(s, "\\", "\\\\")
	r = strings.ReplaceAll(r, "(", "\\(")
	r = strings.ReplaceAll(r, ")", "\\)")
	r = strings.ReplaceAll(r, "\r", "\\r")
	return r
}

func (p *Fpdf) textString(s string) string {
	if !isASCII(s) {
		s = utf8ToUTF16BEWithBOM(s)
	}
	return "(" + p.escape(s) + ")"
}

func (p *Fpdf) doUnderline(x, y float64, txt string) string {
	if p.currentFont == nil {
		return ""
	}
	w := p.GetStringWidth(txt) + p.ws*float64(strings.Count(txt, " "))
	return sprintf("%.2F %.2F %.2F %.2F re f", x*p.k, (p.h-(y-p.currentFont.up/1000*p.fontSize))*p.k, w*p.k, -p.currentFont.ut/1000*p.fontSizePt)
}

func (p *Fpdf) parseImageFile(file string) *pdfImage {
	f, err := os.Open(file)
	if err != nil {
		p.panicError("can't open image file: " + file)
	}
	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		p.panicError("missing or incorrect image file: " + file)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		p.panicError("unable to seek image file")
	}

	switch strings.ToLower(format) {
	case "jpeg":
		data, readErr := io.ReadAll(f)
		if readErr != nil {
			p.panicError("unable to read JPEG image file")
		}
		return &pdfImage{w: cfg.Width, h: cfg.Height, cs: "DeviceRGB", bpc: 8, f: "DCTDecode", data: data}
	default:
		img, _, decodeErr := image.Decode(f)
		if decodeErr != nil {
			p.panicError("unable to decode image file: " + file)
		}

		var encoded bytes.Buffer
		if encodeErr := stdjpeg.Encode(&encoded, img, &stdjpeg.Options{Quality: 90}); encodeErr != nil {
			p.panicError("unable to encode image as JPEG: " + file)
		}

		return &pdfImage{w: cfg.Width, h: cfg.Height, cs: "DeviceRGB", bpc: 8, f: "DCTDecode", data: encoded.Bytes()}
	}
}

func (p *Fpdf) charWidth(c byte) int {
	if p.currentFont == nil {
		return 0
	}
	w := p.currentFont.cw[c]
	if w == 0 {
		return p.currentFont.cw['?']
	}
	return w
}

func (p *Fpdf) loadFontAsset(file string) (*pdfFont, bool) {
	key := strings.ToLower(filepath.Base(file))
	f, ok := p.assetFonts[key]
	if !ok {
		return nil, false
	}
	return f, true
}

// HTML rendering support structures
type pdfHTMLStyle struct {
	colorR, colorG, colorB float64
	fontFamily             string
	fontStyle              string
	fontSize               float64
	colorSet               bool
}

type pdfHTMLState struct {
	p *Fpdf

	boldCount      int
	italicCount    int
	underlineCount int
	href           string
	pre            bool

	tableBorder int
	tdBegin     bool
	thBegin     bool
	tdWidth     float64
	tdHeight    float64
	tdAlign     string
	tdBgColor   bool
	trBgColor   bool
	cellPadding float64
	cellSpacing float64

	inTable        bool
	inRow          bool
	cellText       string
	colIndex       int
	tableColWidths map[int]float64
	rowStartY      float64
	maxRowHeight   float64
	tdWidthAttr    string

	tdColorR, tdColorG, tdColorB float64
	tdColorSet                   bool

	styleStack []pdfHTMLStyle

	fontSet  bool
	colorSet bool

	listDepth int
	listType  string
	listCount int
	listStack []pdfHTMLListState
	currAlign string

	defaultFontSize float64
	scriptActive    bool
	scriptDeltaY    float64
}

type pdfHTMLListState struct {
	listType  string
	listCount int
}

func (s *pdfHTMLState) renderHTML(input string) {
	tagRe := regexp.MustCompile(`(?is)<[^>]+>`)
	segments := tagRe.FindAllStringIndex(input, -1)
	pos := 0
	for _, seg := range segments {
		if seg[0] > pos {
			s.handleText(input[pos:seg[0]])
		}
		s.handleTag(input[seg[0]:seg[1]])
		pos = seg[1]
	}
	if pos < len(input) {
		s.handleText(input[pos:])
	}
}

func (s *pdfHTMLState) handleText(raw string) {
	if raw == "" {
		return
	}
	text := raw
	if !s.pre {
		re := regexp.MustCompile(`\s+`)
		text = re.ReplaceAllString(text, " ")
	}
	text = stdhtml.UnescapeString(text)
	text = normalizeHTMLTextForPDF(text)
	if text == "" {
		return
	}
	if s.href != "" {
		s.putLink(s.href, text)
		return
	}
	if s.tdBegin || s.thBegin {
		s.cellText += text
		return
	}
	if (s.inTable || s.inRow) && strings.TrimSpace(text) == "" {
		return
	}
	s.p.Write(5, text, "")
}

func (s *pdfHTMLState) handleTag(rawTag string) {
	tagContent := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(rawTag, "<"), ">"))
	if tagContent == "" {
		return
	}
	isClosing := strings.HasPrefix(tagContent, "/")
	isSelfClosing := strings.HasSuffix(tagContent, "/")
	if isClosing {
		tagName := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(tagContent, "/")))
		s.closeTag(tagName)
		return
	}
	if isSelfClosing {
		tagContent = strings.TrimSpace(strings.TrimSuffix(tagContent, "/"))
	}
	tagName, attrs := parseHTMLTag(tagContent)
	if tagName == "" {
		return
	}
	s.openTag(strings.ToUpper(tagName), attrs)
	if isSelfClosing {
		s.closeTag(strings.ToUpper(tagName))
	}
}

func (s *pdfHTMLState) openTag(tag string, attrs map[string]string) {
	if style, ok := attrs["STYLE"]; ok {
		css := parseCSSStyle(style)
		if color, ok := css["color"]; ok {
			r, g, b := htmlColorToRGB(color)
			s.p.SetTextColor(float64(r), float64(g), float64(b))
			s.colorSet = true
		}
		if bgColor, ok := css["background-color"]; ok {
			r, g, b := htmlColorToRGB(bgColor)
			s.p.SetFillColor(float64(r), float64(g), float64(b))
			s.tdBgColor = true
		}
	}
	switch tag {
	case "STRONG", "B":
		s.setStyle("B", true)
	case "EM", "I":
		s.setStyle("I", true)
	case "U":
		s.setStyle("U", true)
	case "BR":
		s.p.Ln(5)
	case "P", "DIV":
		s.p.Ln(5)
	case "A":
		s.href = attrs["HREF"]
		s.p.SetTextColor(0, 0, 255)
		s.setStyle("U", true)
	}
}

func (s *pdfHTMLState) closeTag(tag string) {
	switch tag {
	case "STRONG", "B":
		s.setStyle("B", false)
	case "EM", "I":
		s.setStyle("I", false)
	case "U":
		s.setStyle("U", false)
	case "A":
		s.href = ""
		s.setStyle("U", false)
		s.p.SetTextColor(0, math.NaN(), math.NaN())
	}
}

func (s *pdfHTMLState) setStyle(tag string, enable bool) {
	switch tag {
	case "B":
		if enable {
			s.boldCount++
		} else if s.boldCount > 0 {
			s.boldCount--
		}
	case "I":
		if enable {
			s.italicCount++
		} else if s.italicCount > 0 {
			s.italicCount--
		}
	case "U":
		if enable {
			s.underlineCount++
		} else if s.underlineCount > 0 {
			s.underlineCount--
		}
	}
	style := ""
	if s.boldCount > 0 {
		style += "B"
	}
	if s.italicCount > 0 {
		style += "I"
	}
	if s.underlineCount > 0 {
		style += "U"
	}
	s.p.SetFont("", style, 0)
}

func (s *pdfHTMLState) putLink(url, text string) {
	s.p.SetTextColor(0, 0, 255)
	s.setStyle("U", true)
	s.p.Write(5, text, url)
	s.setStyle("U", false)
	s.p.SetTextColor(0, math.NaN(), math.NaN())
}

// Utility functions
func sprintf(format string, args ...interface{}) string { return fmt.Sprintf(format, args...) }
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}
func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func flateCompress(data []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, _ = w.Write(data)
	_ = w.Close()
	return b.Bytes()
}
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}
func latin1ToUTF8(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 128 {
			b.WriteByte(0xC0 | (c >> 6))
			b.WriteByte(0x80 | (c & 0x3F))
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}
func utf8ToUTF16BEWithBOM(s string) string {
	runes := []rune(s)
	buf := make([]byte, 2, 2+len(runes)*2)
	buf[0] = 0xFE
	buf[1] = 0xFF
	for _, r := range runes {
		if r > 0xFFFF {
			r = '?'
		}
		tmp := make([]byte, 2)
		binary.BigEndian.PutUint16(tmp, uint16(r))
		buf = append(buf, tmp...)
	}
	return string(buf)
}
func normalizeHTMLTextForPDF(text string) string {
	if text == "" {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if r >= 0 && r <= 255 {
			b.WriteByte(byte(r))
		} else {
			b.WriteByte('?')
		}
	}
	return b.String()
}
func parseHTMLTag(content string) (string, map[string]string) {
	attrs := map[string]string{}
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", attrs
	}
	tagName := parts[0]
	rest := ""
	if len(content) > len(tagName) {
		rest = strings.TrimSpace(content[len(tagName):])
	}
	attrRe := regexp.MustCompile(`(?is)([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*("([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	matches := attrRe.FindAllStringSubmatch(rest, -1)
	for _, m := range matches {
		key := strings.ToUpper(strings.TrimSpace(m[1]))
		val := ""
		if m[3] != "" {
			val = m[3]
		} else if m[4] != "" {
			val = m[4]
		} else {
			val = m[5]
		}
		attrs[key] = val
	}
	return tagName, attrs
}
func parseCSSStyle(style string) map[string]string {
	styles := map[string]string{}
	parts := strings.Split(style, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			styles[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
		}
	}
	return styles
}
func htmlColorToRGB(color string) (int, int, int) {
	return 0, 0, 0 // Simplified for brevity
}
