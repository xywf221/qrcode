package qrcode

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

type goldenCase struct {
	text    string
	level   Level
	version int
	mask    int
	hash    string
}

// goldenCases were produced by this encoder after it was verified module-for-
// module against rsc.io/qr and github.com/boombuler/barcode. They guard against
// regressions in the encoder pipeline.
func goldenCases() []goldenCase {
	return []goldenCase{
		{"HELLO", Low, 1, 2, "881c42d5215ce1be9ea181cb671cc8ae02c7217438f8b58116801807d3529388"},
		{"HELLO", Medium, 1, 2, "e0316904b17d6e381368946c2bd9ed1a01f997a6df23f85252bdd1f76ca90040"},
		{"HELLO", Quartile, 1, 2, "7db751dd19962e3f71942551f4d53c359f18957167e72effa297e33a7b492b35"},
		{"HELLO", High, 1, 3, "ec6a5502873de7f01ae8c0e68571210051c063fec596fe53ae95e623ebf25f36"},
		{"hello world", Low, 1, 0, "8ceb8d675fa0125c3c4709f175350a322718f349cc2887074da14cd6d863b2c3"},
		{"hello world", Medium, 1, 2, "c95b298d9ca68ddf3fb7ca1544cc83d3f141dcb374b5ae9e0228df1f689765a5"},
		{"hello world", Quartile, 1, 2, "3046835229dcc4edd21d9ca158ef187796aed052a30a0e4869e733a87086ecf9"},
		{"hello world", High, 2, 2, "00f7571b28789acb339430785adcd9dfe7951933b6efb02d6af83f346fe47856"},
		{"https://example.com", Low, 2, 6, "224ffd1fd69b7db2dba5207989234944f0a032dd0626e8b43d3448b6c759e583"},
		{"https://example.com", Medium, 2, 1, "8bf2dcbeeedba5c1dc6cdaa85b40ef3401a32bfbd11d83c10dac3dbc0db23926"},
		{"https://example.com", Quartile, 2, 0, "15b61cfb4762b24c35f35c60f4666ddeeb449d1fae69e1e06898eb05133f7b21"},
		{"https://example.com", High, 3, 3, "63c2a02d7c1d810b4bef1b3bdedab1116dedb209d2743073c6f5fb91ed3d475d"},
		{"ABCDEFG 12345 ", Low, 1, 7, "ef67741dd1c7152f17bcc53cef4e5e77a1bb5b0233d4983c3ca8a73574044524"},
		{"ABCDEFG 12345 ", Medium, 1, 0, "30f8a9f1df3aecc76c909cb89ee8da42fbec454cecefad4f535b4ba05285a5c7"},
		{"ABCDEFG 12345 ", Quartile, 1, 0, "dfe6c7ff72924f27fe4b0807ad7d66039e4fd54f2014e6473090bc4c338ebce0"},
		{"ABCDEFG 12345 ", High, 2, 6, "087b2a91501af990e97d08d1d975f2bd31de27618b4e85e54911fcc89af3c94c"},
		{"abc def abc def abc def abc def abc def abc def abc def abc def ", Low, 4, 2, "54412451b35b861563c4045edfeca135c1b591ec52f4749054c13c1dfda91414"},
		{"abc def abc def abc def abc def abc def abc def abc def abc def ", Medium, 5, 6, "09bd54f4271fe85559ddcb0fd63ec37003672abd359822cbade6c5bb3f0b841d"},
		{"abc def abc def abc def abc def abc def abc def abc def abc def ", Quartile, 6, 4, "9fed774d33e9793af14bba6f0eb2db0859bb7a3cf1fbc969ffb3406f3c42709e"},
		{"abc def abc def abc def abc def abc def abc def abc def abc def ", High, 7, 4, "b852b92234bae674b59d1495f95f7c84ba6f68d944aaf92823ab78b0a1e83dfe"},
	}
}

func TestGolden(t *testing.T) {
	for _, g := range goldenCases() {
		q, err := NewString(g.text, WithLevel(g.level), WithBoostECL(false))
		if err != nil {
			t.Fatalf("%q %v: %v", g.text, g.level, err)
		}
		if q.Version() != g.version {
			t.Errorf("%q %v: version = %d, want %d", g.text, g.level, q.Version(), g.version)
		}
		if q.Mask() != g.mask {
			t.Errorf("%q %v: mask = %d, want %d", g.text, g.level, q.Mask(), g.mask)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(q.Bitmap())); got != g.hash {
			t.Errorf("%q %v: module hash = %s, want %s", g.text, g.level, got, g.hash)
		}
	}
}
