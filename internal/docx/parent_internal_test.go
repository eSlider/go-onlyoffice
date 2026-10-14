package docx

import "testing"

func TestCommentsParentResolution(t *testing.T) {
	f := &File{
		commentsName: "word/comments.xml",
		comments: `<?xml version="1.0"?><w:comments xmlns:w="x">` +
			`<w:comment w:id="0" w:author="A"><w:p w14:paraId="00000001"><w:r><w:t>q</w:t></w:r></w:p></w:comment>` +
			`<w:comment w:id="1" w:author="B"><w:p w14:paraId="00000002"><w:r><w:t>a</w:t></w:r></w:p></w:comment>` +
			`</w:comments>`,
	}
	f.parts = map[string][]byte{
		partCommentsExt: []byte(`<w15:commentsEx>` +
			`<w15:commentEx w15:paraId="00000001" w15:done="0"/>` +
			`<w15:commentEx w15:paraId="00000002" w15:paraIdParent="00000001" w15:done="0"/>` +
			`</w15:commentsEx>`),
	}
	cs, err := f.Comments()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 {
		t.Fatalf("want 2, got %d", len(cs))
	}
	for _, c := range cs {
		t.Logf("id=%s parent=%q text=%q", c.ID, c.ParentID, c.Text)
	}
	if cs[1].ParentID != "0" {
		t.Fatalf("reply parent = %q, want 0", cs[1].ParentID)
	}
}
