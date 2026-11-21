package transfer

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Resource struct { URI, Key, Hash string }

// ParseResource accepts download links only; it never fetches a URL.
func ParseResource(raw string) (Resource, error) {
	raw = strings.TrimSpace(raw)
	if strings.ContainsAny(raw,"\r\n\x00") { return Resource{},fmt.Errorf("请一次提交一个有效下载链接") }
	if len(raw) > 16384 { return Resource{}, fmt.Errorf("下载链接过长") }
	if strings.HasPrefix(strings.ToLower(raw),"ed2k://") {
		parts := strings.Split(raw, "|")
		if len(parts) == 6 && parts[1] == "file" && parts[2] != "" && parts[5] == "/" {
			size, sizeErr := strconv.ParseInt(parts[3], 10, 64)
			hash, hashErr := hex.DecodeString(parts[4])
			if sizeErr == nil && size > 0 && hashErr == nil && len(hash) == 16 {
				parts[0],parts[3],parts[4] = "ed2k://",strconv.FormatInt(size,10),hex.EncodeToString(hash)
				return Resource{URI: strings.Join(parts,"|"), Key: "ed2k:"+parts[4]+":"+parts[3], Hash: parts[4]}, nil
			}
		}
		return Resource{},fmt.Errorf("ed2k 文件链接无效")
	}
	u, err := url.Parse(raw)
	if err != nil { return Resource{}, fmt.Errorf("下载链接无效") }
	if strings.EqualFold(u.Scheme, "magnet") {
		var hash string
		for _, xt := range u.Query()["xt"] {
			if !strings.HasPrefix(strings.ToLower(xt), "urn:btih:") { continue }
			value := xt[len("urn:btih:"):]
			var decoded []byte
			if len(value) == 32 { decoded, err = base32.StdEncoding.DecodeString(strings.ToUpper(value))
			} else { decoded, err = hex.DecodeString(value) }
			if err != nil || len(decoded) != 20 { return Resource{}, fmt.Errorf("磁力哈希无效") }
			candidate := hex.EncodeToString(decoded)
			if hash != "" && hash != candidate { return Resource{}, fmt.Errorf("磁力链接包含多个不同哈希") }
			hash = candidate
		}
		if hash != "" { return Resource{URI: "magnet:?xt=urn:btih:"+hash, Key: hash, Hash: hash}, nil }
	}
	return Resource{}, fmt.Errorf("仅支持有效的 magnet 或 ed2k 文件链接")
}
