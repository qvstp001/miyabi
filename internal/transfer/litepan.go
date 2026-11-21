package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ppxb/miyabi/internal/netx"
)

// LitePan's open automation endpoint accepts task API keys. Admin-only STRM
// endpoints do not. A matching webhook rule must run the selected STRM task.
func pushLitePan(ctx context.Context,c Config) error {
	if err := validateConfig(c); err != nil { return err }
	body,_ := json.Marshal(map[string]string{"event":EventName,"source":"miyabi","path":c.Path})
	req,err := http.NewRequestWithContext(ctx,http.MethodPost,c.LitePanURL+"/api/open/automation/events",bytes.NewReader(body))
	if err != nil { return err }
	req.Header.Set("Content-Type","application/json")
	req.Header.Set("Authorization","Bearer "+c.APIKey)
	transport:=netx.NewTransport(nil)
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport:transport,Timeout:30*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
	response,err := client.Do(req)
	if err != nil { return fmt.Errorf("LitePan 推送失败，请检查连接") }
	defer response.Body.Close()
	var result struct {
		Success bool `json:"success"`
		Data struct { Matched int `json:"matched"` } `json:"data"`
	}
	if response.StatusCode != http.StatusOK { return fmt.Errorf("LitePan 返回 HTTP %d",response.StatusCode) }
	if err := json.NewDecoder(io.LimitReader(response.Body,1<<20)).Decode(&result); err != nil { return fmt.Errorf("LitePan 返回格式无效") }
	if !result.Success || result.Data.Matched <= 0 { return fmt.Errorf("LitePan 未触发任务，请配置事件 %s 的 STRM 自动化规则",EventName) }
	return nil
}
