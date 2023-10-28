package main

type (
	Record struct {
		Project string
		Hours   string
		Year    uint16
		Month   uint16
		ID      uint32
		UpAt    int64
		DownAt  int64
	}

	Config struct {
		DB      string `yaml:"db"`
		Project string `yaml:"project"`
		Cursor  uint32 `yaml:"cursor"`
	}
)
