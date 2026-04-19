package document

import "fmt"

func RenderDocument(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	return doc.Raw, nil
}
