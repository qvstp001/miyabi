package transfer

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/ppxb/miyabi/internal/database"
	"github.com/ppxb/miyabi/internal/domain"
	"github.com/ppxb/miyabi/internal/drive"
)

const configKey = "browser.downloads"
const EventName = "miyabi.download.completed"

type Config struct {
	LitePanURL string `json:"litepan_url"`
	APIKey string `json:"api_key,omitempty"`
	Path string `json:"path"`
	HasAPIKey bool `json:"has_api_key"`
	AccountID string `json:"account_id"`
	DirectoryID string `json:"directory_id"`
}

func (s *Service) config(ctx context.Context) (Config,error) {
	value,_,err := database.LoadSetting[Config](ctx,s.db,configKey)
	return value,err
}

func (s *Service) Config(ctx context.Context) (Config,error) {
	c,err := s.config(ctx)
	c.HasAPIKey,c.APIKey = c.APIKey != "",""
	return c,err
}

func validateConfig(c Config) error {
	u,err := url.Parse(c.LitePanURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("请填写有效的 LitePan HTTP 地址")
	}
	if c.APIKey == "" || strings.ContainsAny(c.APIKey,"\r\n") { return fmt.Errorf("请配置 LitePan 任务 API Key") }
	if !strings.HasPrefix(c.Path,"/") || path.Clean(c.Path) != c.Path || strings.ContainsAny(c.Path,"\r\n") { return fmt.Errorf("请填写 LitePan 中对应下载目录的绝对路径") }
	return nil
}

func (s *Service) SaveConfig(ctx context.Context,c Config) (Config,error) {
	if err := s.mu.Lock(ctx); err != nil { return Config{}, err }
	defer s.mu.Unlock()
	c.LitePanURL,c.Path,c.APIKey = strings.TrimRight(strings.TrimSpace(c.LitePanURL),"/"),strings.TrimSpace(c.Path),strings.TrimSpace(c.APIKey)
	old,err := s.config(ctx)
	if err != nil { return Config{},err }
	if c.APIKey == "" && c.LitePanURL == old.LitePanURL { c.APIKey = old.APIKey }
	if err := validateConfig(c); err != nil { return Config{},domain.E(domain.KindInvalid,err.Error(),nil) }
	sess,err:=s.drive.OpenFiles(ctx)
	if err!=nil { return Config{},err }
	source:=sess.Source()
	if c.AccountID != source.AccountID || c.DirectoryID != source.Directory.ID { return Config{},drive.ErrSourceChanged }
	c.AccountID,c.DirectoryID=source.AccountID,source.Directory.ID
	c.HasAPIKey = false
	if err := sess.WithSource(ctx,func()error{return database.SaveSetting(ctx,s.db,configKey,c)}); err != nil { return Config{},err }
	return s.Config(ctx)
}
