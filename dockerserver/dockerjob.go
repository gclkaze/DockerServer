package dockerserver

type DockerJob struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"createdA"`
	Admin        bool   `json:"admin"`
	Image        string `json:"image"`
	LocalVolumes string `json:"localVolumes"`
	Mounts       string `json:"mounts"`
	Names        string `json:"names"`
	Networks     string `json:"networks"`
	Ports        string `json:"ports"`
	RunningFor   string `json:"runningFor"`
	Size         string `json:"size"`
	State        string `json:"state"`
	Status       string `json:"status"`
	SocketNumber string
}
