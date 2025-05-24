package mindscape

// Config 是 mindscape 的配置
type Config struct {
	Host string `json:"host" toml:"host" yaml:"host"`
	Port int    `json:"port" toml:"port" yaml:"port"`
}
