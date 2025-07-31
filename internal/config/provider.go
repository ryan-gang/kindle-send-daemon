package config

// ConfigProvider defines the interface for configuration access
type ConfigProvider interface {
	GetSender() string
	GetReceiver() string
	GetStorePath() string
	GetPassword() string
	GetServer() string
	GetPort() int
	GetBookmarkPath() string
	GetCheckInterval() int
	IsDaemonEnabled() bool
	GetLogPath() string
	GetPidFile() string
}

// ConfigImpl implements ConfigProvider interface
type ConfigImpl struct {
	cfg *config
}

// NewConfigProvider creates a new ConfigProvider instance
func NewConfigProvider(cfg *config) ConfigProvider {
	return &ConfigImpl{cfg: cfg}
}

func (c *ConfigImpl) GetSender() string {
	return c.cfg.Sender
}

func (c *ConfigImpl) GetReceiver() string {
	return c.cfg.Receiver
}

func (c *ConfigImpl) GetStorePath() string {
	return c.cfg.StorePath
}

func (c *ConfigImpl) GetPassword() string {
	return c.cfg.Password
}

func (c *ConfigImpl) GetServer() string {
	return c.cfg.Server
}

func (c *ConfigImpl) GetPort() int {
	return c.cfg.Port
}

func (c *ConfigImpl) GetBookmarkPath() string {
	return c.cfg.BookmarkPath
}

func (c *ConfigImpl) GetCheckInterval() int {
	return c.cfg.CheckInterval
}

func (c *ConfigImpl) IsDaemonEnabled() bool {
	return c.cfg.DaemonEnabled
}

func (c *ConfigImpl) GetLogPath() string {
	return c.cfg.LogPath
}

func (c *ConfigImpl) GetPidFile() string {
	return c.cfg.PidFile
}
