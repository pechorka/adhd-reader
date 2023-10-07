package epub

type container struct {
	Rootfiles []rootfile `xml:"rootfiles>rootfile"`
}

func (c container) ContentFilePath() (string, bool) {
	for _, r := range c.Rootfiles {
		if r.MediaType == "application/oebps-package+xml" {
			return r.FullPath, true
		}
	}
	return "", false
}

type rootfile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

type opf struct {
	Metadata opfMetadata `xml:"metadata"`
	Manifest []manifest  `xml:"manifest>item"`
	Spine    spine       `xml:"spine"`
	Guide    []guide     `xml:"guide>reference"`
}

type opfMetadata struct {
	Title       string `xml:"title"`
	Language    string `xml:"language"`
	Identifier  string `xml:"identifier"`
	Contributor string `xml:"contributor"`
	Creator     string `xml:"creator"`
	Description string `xml:"description"`
	Publisher   string `xml:"publisher"`
	Subject     string `xml:"subject"`
	Type        string `xml:"type"`
	Source      string `xml:"source"`
	Relation    string `xml:"relation"`
	Rights      string `xml:"rights"`
	Date        string `xml:"date"`
	Format      string `xml:"format"`
	Coverage    string `xml:"coverage"`
}

type manifest struct {
	Id        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

type spine struct {
	Toc      string    `xml:"toc,attr"`
	ItemRefs []itemref `xml:"itemref"`
}

type itemref struct {
	Idref string `xml:"idref,attr"`
}

type guide struct {
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
	Href  string `xml:"href,attr"`
}
