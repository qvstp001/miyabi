package transfer

import (
	"strings"
	"testing"
)

func TestParseResource(t *testing.T) {
	hash := strings.Repeat("a", 40)
	for _,uri := range []string{"magnet:?xt=urn:btih:"+hash,"magnet:?dn=movie&xt=urn:btih:"+strings.ToUpper(hash)} {
		r,err:=ParseResource(uri)
		if err!=nil || r.Key!=hash || r.URI!="magnet:?xt=urn:btih:"+hash { t.Fatalf("parse %q: %+v %v",uri,r,err) }
	}
	base32,err:=ParseResource("magnet:?xt=urn:btih:"+strings.Repeat("A",32))
	if err!=nil || base32.Key!=strings.Repeat("0",40){t.Fatalf("base32: %+v %v",base32,err)}
	ed2k,err:=ParseResource("ed2k://|file|movie%20part.mkv|123456|"+strings.Repeat("B",32)+"|/")
	if err!=nil || ed2k.Key!="ed2k:"+strings.Repeat("b",32)+":123456"{t.Fatalf("ed2k: %+v %v",ed2k,err)}
	for _,invalid:=range []string{"https://example.com/movie","magnet:?xt=urn:btmh:bad","magnet:?xt=urn:btih:bad","ed2k://|file|movie|0|"+strings.Repeat("a",32)+"|/","ed2k://|server|example|80|/","magnet:?xt=urn:btih:"+hash+"&xt=urn:btih:"+strings.Repeat("b",40)}{
		if _,err:=ParseResource(invalid);err==nil{t.Errorf("accepted %q",invalid)}
	}
}
