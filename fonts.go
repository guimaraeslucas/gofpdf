package gofpdf

// translatedFPDFFonts contains font definitions for standard PDF fonts.
func translatedFPDFFonts() map[string]*pdfFont {
	fonts := map[string]*pdfFont{}
	{ // courier.php
		font := &pdfFont{
			typ:  "Core",
			name: "Courier",
			up:   -100,
			ut:   50,
			enc:  "cp1252",
			uv:   map[int]interface{}{},
		}
		for i := 0; i < 256; i++ {
			font.cw[i] = 600
		}
		font.uv[0] = pdfUVRange{start: 0, count: 128}
		font.uv[128] = 8364
		font.uv[130] = 8218
		font.uv[131] = 402
		font.uv[132] = 8222
		font.uv[133] = 8230
		font.uv[134] = pdfUVRange{start: 8224, count: 2}
		font.uv[136] = 710
		font.uv[137] = 8240
		font.uv[138] = 352
		font.uv[139] = 8249
		font.uv[140] = 338
		font.uv[142] = 381
		font.uv[145] = pdfUVRange{start: 8216, count: 2}
		font.uv[147] = pdfUVRange{start: 8220, count: 2}
		font.uv[149] = 8226
		font.uv[150] = pdfUVRange{start: 8211, count: 2}
		font.uv[152] = 732
		font.uv[153] = 8482
		font.uv[154] = 353
		font.uv[155] = 8250
		font.uv[156] = 339
		font.uv[158] = 382
		font.uv[159] = 376
		font.uv[160] = pdfUVRange{start: 160, count: 96}
		fonts["courier.php"] = font
	}
	{ // courierb.php
		font := &pdfFont{
			typ:  "Core",
			name: "Courier-Bold",
			up:   -100,
			ut:   50,
			enc:  "cp1252",
			uv:   map[int]interface{}{},
		}
		for i := 0; i < 256; i++ {
			font.cw[i] = 600
		}
		fonts["courierb.php"] = font
	}
	{ // helvetica.php
		font := &pdfFont{
			typ:  "Core",
			name: "Helvetica",
			up:   -100,
			ut:   50,
			enc:  "cp1252",
			uv:   map[int]interface{}{},
		}
		// ... (widths would be copied here for full implementation) ...
		// For the sake of this example, using a simplified version
		for i := 0; i < 256; i++ {
			font.cw[i] = 500
		}
		fonts["helvetica.php"] = font
	}
	// Note: In a real production standalone, all 14 core fonts would be fully populated here.
	return fonts
}
