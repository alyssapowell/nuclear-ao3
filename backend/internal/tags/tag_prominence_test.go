package tags

import "testing"

// Tag prominence (which tags are central to a work) is only a schema so far:
// no Go code reads or writes it, and migration 010's trigger and the old
// tests joined work_tags on a tag_name column that work_tags (work_id, tag_id)
// does not have. Listed in docs/FEATURES.md.
func TestTagProminence(t *testing.T) {
	t.Skip("bug: tag prominence is not built; migration 010 refers to work_tags.tag_name, which does not exist (see docs/FEATURES.md)")
}
