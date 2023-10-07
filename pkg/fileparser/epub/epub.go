package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	containerPath = "META-INF/container.xml"
)

var (
	ErrNoContainer = errors.New("no container.xml found")
)

func PlainText(data []byte) (string, error) {
	allFiles, err := parseAllFiles(data)
	if err != nil {
		return "", err
	}

	container, err := parseContainer(allFiles)
	if err != nil {
		return "", err
	}

	content, err := parseContent(allFiles, container)
	if err != nil {
		return "", err
	}

	htmlFiles, err := allHtmlFiles(allFiles, content)
	if err != nil {
		return "", err
	}

	orderedHtmls := orderHtmlFiles(content, htmlFiles)
	return plainTextFromHtmls(orderedHtmls)
}

func parseAllFiles(data []byte) (map[string]*zip.File, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	fileMap := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	return fileMap, nil
}

func parseContainer(files allFiles) (c container, err error) {
	err = files.decodeFile(containerPath, func(r io.Reader) error {
		return xml.NewDecoder(r).Decode(&c)
	})
	return c, err
}

func parseContent(files allFiles, c container) (opf opf, err error) {
	contentPath, ok := c.ContentFilePath()
	if !ok {
		return opf, errors.New("no content file found")
	}

	err = files.decodeFile(contentPath, func(r io.Reader) error {
		return xml.NewDecoder(r).Decode(&opf)
	})
	return opf, err
}

func allHtmlFiles(files allFiles, o opf) (map[string]string, error) {
	htmlFiles := make(map[string]string, len(o.Manifest)) // map[id]htmlContent
	var b bytes.Buffer
	for _, m := range o.Manifest {
		if m.MediaType != "application/xhtml+xml" {
			continue
		}
		err := files.decodeFile(m.Href, func(r io.Reader) error {
			_, err := io.Copy(&b, r)
			return err
		})
		if err != nil {
			return nil, err
		}
		htmlFiles[m.Id] = b.String()
		b.Reset() // reuse buffer
	}
	return htmlFiles, nil
}

func orderHtmlFiles(o opf, htmlFiles map[string]string) []string {
	var ordered []string
	for _, i := range o.Spine.ItemRefs {
		if html, ok := htmlFiles[i.Idref]; ok {
			ordered = append(ordered, html)
		}
	}
	return ordered
}

func plainTextFromHtmls(htmls []string) (string, error) {
	var totalSize int
	for _, h := range htmls {
		totalSize += len(h)
	}

	var b bytes.Buffer
	b.Grow(totalSize)
	for _, h := range htmls {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(h))
		if err != nil {
			return "", err
		}
		b.WriteString(doc.Text())
		b.WriteByte('\n')
	}

	return b.String(), nil
}

type allFiles map[string]*zip.File

func (a allFiles) decodeFile(path string, decoder func(r io.Reader) error) error {
	f, ok := a[path]
	if !ok {
		return errors.New("file not found")
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}

	if err = decoder(rc); err != nil {
		return err
	}

	return rc.Close()
}
