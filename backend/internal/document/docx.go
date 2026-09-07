package document

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"strings"
)

func GenerateDOCX(title, body string) ([]byte, error) {
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	files := map[string]string{
		"[Content_Types].xml":          contentTypesXML(),
		"_rels/.rels":                  relsXML(),
		"word/document.xml":            documentXML(title, body),
		"word/_rels/document.xml.rels": documentRelsXML(),
		"word/styles.xml":              stylesXML(),
	}
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, fmt.Errorf("create zip entry %s: %w", name, err)
		}
		if _, err := w.Write([]byte(data)); err != nil {
			return nil, fmt.Errorf("write zip entry %s: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return b.Bytes(), nil
}

func contentTypesXML() string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n" +
		"<Types xmlns=\"http://schemas.openxmlformats.org/package/2006/content-types\">\n" +
		"  <Default Extension=\"rels\" ContentType=\"application/vnd.openxmlformats-package.relationships+xml\"/>\n" +
		"  <Default Extension=\"xml\" ContentType=\"application/xml\"/>\n" +
		"  <Override PartName=\"/word/document.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml\"/>\n" +
		"  <Override PartName=\"/word/styles.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml\"/>\n" +
		"</Types>"
}

func relsXML() string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n" +
		"<Relationships xmlns=\"http://schemas.openxmlformats.org/package/2006/relationships\">\n" +
		"  <Relationship Id=\"rId1\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument\" Target=\"word/document.xml\"/>\n" +
		"</Relationships>"
}

func documentRelsXML() string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n" +
		"<Relationships xmlns=\"http://schemas.openxmlformats.org/package/2006/relationships\">\n" +
		"  <Relationship Id=\"rId1\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles\" Target=\"styles.xml\"/>\n" +
		"</Relationships>"
}

func stylesXML() string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n" +
		"<w:styles xmlns:w=\"http://schemas.openxmlformats.org/wordprocessingml/2006/main\">\n" +
		"  <w:docDefaults>\n" +
		"    <w:rPrDefault>\n" +
		"      <w:rPr>\n" +
		"        <w:rFonts w:ascii=\"Arial\" w:hAnsi=\"Arial\"/>\n" +
		"        <w:sz w:val=\"22\"/>\n" +
		"      </w:rPr>\n" +
		"    </w:rPrDefault>\n" +
		"  </w:docDefaults>\n" +
		"  <w:style w:type=\"paragraph\" w:styleId=\"Title\">\n" +
		"    <w:name w:val=\"Title\"/>\n" +
		"    <w:basedOn w:styleId=\"Normal\"/>\n" +
		"    <w:rPr>\n" +
		"      <w:b/>\n" +
		"      <w:sz w:val=\"32\"/>\n" +
		"    </w:rPr>\n" +
		"  </w:style>\n" +
		"  <w:style w:type=\"paragraph\" w:styleId=\"Normal\">\n" +
		"    <w:name w:val=\"Normal\"/>\n" +
		"  </w:style>\n" +
		"</w:styles>"
}

func documentXML(title, body string) string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n" +
		"<w:document xmlns:w=\"http://schemas.openxmlformats.org/wordprocessingml/2006/main\">\n" +
		"  <w:body>\n" +
		"    <w:p>\n" +
		"      <w:pPr>\n" +
		"        <w:pStyle w:val=\"Title\"/>\n" +
		"      </w:pPr>\n" +
		"      <w:r>\n" +
		"        <w:t>" + xmlEscape(title) + "</w:t>\n" +
		"      </w:r>\n" +
		"    </w:p>\n" +
		"    <w:p>\n" +
		"      <w:r>\n" +
		"        <w:t>" + xmlEscape(body) + "</w:t>\n" +
		"      </w:r>\n" +
		"    </w:p>\n" +
		"    <w:sectPr>\n" +
		"      <w:pgSz w:w=\"12240\" w:h=\"15840\"/>\n" +
		"      <w:pgMar w:top=\"1440\" w:right=\"1440\" w:bottom=\"1440\" w:left=\"1440\"/>\n" +
		"    </w:sectPr>\n" +
		"  </w:body>\n" +
		"</w:document>"
}

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&",
		"<", "<",
		">", ">",
	)
	return r.Replace(s)
}

func ReadTextTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
