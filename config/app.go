package config

type UIConfig struct {
	Enabled bool `config:"enabled"`
	LocalPath    string `config:"path"`
	LocalEnabled bool   `config:"local"`
	VFSEnabled   bool   `config:"vfs"`
}

type AppConfig struct {
	UI UIConfig `config:"ui"`
}
