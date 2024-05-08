package prop

import (
	"bytes"
	"encoding/xml"
)

// PropertyXML represents a single DAV resource property as defined in RFC 4918.
// http://www.webdav.org/specs/rfc4918.html#data.model.for.resource.properties
type PropertyXML struct {
	// XMLName is the fully qualified name that identifies this property.
	XMLName xml.Name

	// Lang is an optional xml:lang attribute.
	Lang string `xml:"xml:lang,attr,omitempty"`

	// InnerXML contains the XML representation of the property value.
	// See http://www.webdav.org/specs/rfc4918.html#property_values
	//
	// Property values of complex type or mixed-content must have fully
	// expanded XML namespaces or be self-contained with according
	// XML namespace declarations. They must not rely on any XML
	// namespace declarations within the scope of the XML document,
	// even including the DAV: namespace.
	InnerXML []byte `xml:",innerxml"`
}

func xmlEscaped(val string) []byte {
	buf := new(bytes.Buffer)
	xml.Escape(buf, []byte(val))
	return buf.Bytes()
}

// Escaped returns a new PropertyXML instance while xml-escaping the value
// TODO properly use the space
func Escaped(key, val string) PropertyXML {
	return PropertyXML{
		XMLName:  xml.Name{Space: "", Local: key},
		Lang:     "",
		InnerXML: xmlEscaped(val),
	}
}

// Raw returns a new PropertyXML instance for the given key/value pair
// TODO properly use the space
func Raw(key, val string) PropertyXML {
	return PropertyXML{
		XMLName:  xml.Name{Space: "", Local: key},
		Lang:     "",
		InnerXML: []byte(val),
	}
}

// Next returns the next token, if any, in the XML stream of d.
// RFC 4918 requires to ignore comments, processing instructions
// and directives.
// http://www.webdav.org/specs/rfc4918.html#property_values
// http://www.webdav.org/specs/rfc4918.html#xml-extensibility
func Next(d *xml.Decoder) (xml.Token, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return t, err
		}
		switch t.(type) {
		case xml.Comment, xml.Directive, xml.ProcInst:
			continue
		default:
			return t, nil
		}
	}
}

// ActiveLock holds active lock xml data
//
//	http://www.webdav.org/specs/rfc4918.html#ELEMENT_activelock
//
// <!ELEMENT activelock (lockscope, locktype, depth, owner?, timeout?,
//
//	locktoken?, lockroot)>
type ActiveLock struct {
	XMLName   xml.Name  `xml:"activelock"`
	Exclusive *struct{} `xml:"lockscope>exclusive,omitempty"`
	Shared    *struct{} `xml:"lockscope>shared,omitempty"`
	Write     *struct{} `xml:"locktype>write,omitempty"`
	Depth     string    `xml:"depth"`
	Owner     Owner     `xml:"owner,omitempty"`
	Timeout   string    `xml:"timeout,omitempty"`
	Locktoken string    `xml:"locktoken>href"`
	Lockroot  string    `xml:"lockroot>href,omitempty"`
}

// Owner captures the inner UML of a lock owner element http://www.webdav.org/specs/rfc4918.html#ELEMENT_owner
type Owner struct {
	InnerXML string `xml:",innerxml"`
}
