package metafields

import (
	shopify "github.com/bold-commerce/go-shopify/v3"
	"sort"
	"strings"
)

type lessFunc func(mf1, mf2 *shopify.Metafield) int

type metafieldsSorter struct {
	metafields []shopify.Metafield
	less       []lessFunc
}

func (ms *metafieldsSorter) Len() int {
	return len(ms.metafields)
}

func (ms *metafieldsSorter) Swap(i, j int) {
	ms.metafields[i], ms.metafields[j] = ms.metafields[j], ms.metafields[i]
}

func (mf *metafieldsSorter) Less(i, j int) bool {
	less := false
	for _, fx := range mf.less {
		order := fx(&mf.metafields[i], &mf.metafields[j])
		if order == 0 {
			less = false
			continue
		}

		less = order == -1
		break
	}

	return less
}

func (ms *metafieldsSorter) Sort(metafields []shopify.Metafield) {
	ms.metafields = metafields
	sort.Sort(ms)
}

func byNamespaceAsc(mf1, mf2 *shopify.Metafield) int {
	return strings.Compare(strings.ToLower(mf1.Namespace), strings.ToLower(mf2.Namespace))
}

func byNamespaceDesc(mf1, mf2 *shopify.Metafield) int {
	return byNamespaceAsc(mf2, mf1)
}

func byKeyAsc(mf1, mf2 *shopify.Metafield) int {
	return strings.Compare(strings.ToLower(mf1.Key), strings.ToLower(mf2.Key))
}

func byKeyDesc(mf1, mf2 *shopify.Metafield) int {
	return byKeyAsc(mf2, mf1)
}

func byCreatedAtAsc(mf1, mf2 *shopify.Metafield) int {
	if mf1.CreatedAt.Before(*mf2.CreatedAt) {
		return -1
	}

	if mf1.CreatedAt.After(*mf2.CreatedAt) {
		return 1
	}

	return 0
}

func byCreatedAtDesc(mf1, mf2 *shopify.Metafield) int {
	return byUpdatedAtAsc(mf2, mf1)
}

func byUpdatedAtAsc(mf1, mf2 *shopify.Metafield) int {
	if mf1.UpdatedAt.Before(*mf2.UpdatedAt) {
		return -1
	}

	if mf1.UpdatedAt.After(*mf2.UpdatedAt) {
		return 1
	}

	return 0
}

func byUpdatedAtDesc(mf1, mf2 *shopify.Metafield) int {
	return byUpdatedAtAsc(mf2, mf1)
}
