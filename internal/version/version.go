package version

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

var current = Info{
	Version: "0.1.0",
	Commit:  "dev",
}

func Set(version, commit string) {
	current = Info{
		Version: version,
		Commit:  commit,
	}
}

func Current() Info {
	return current
}
