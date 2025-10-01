package dockerserver

/*
{"Command":"\"docker-entrypoint.s…\"","CreatedAt":"2024-07-26 14:51:56 +0300 EEST","ID":"d7f7163be952","Image":"postgres","Labels":"",
"LocalVolumes":"1","Mounts":"23cc14937e81dc…","Names":"build-skills-postgres","Networks":"bridge",
"Ports":"0.0.0.0:5432-\u003e5432/tcp","RunningFor":"14 months ago","Size":"0B","State":"running","Status":"Up 11 minutes"}`*/
type DockerJob struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"createdA"`
	Admin        bool   `json:"admin"`
	Image        string `json:"image"`
	LocalVolumes string `json:"localVolumes"`
	Mounts       string `json:"mounts"`
	Names        string `json:"names"`

	Networks string `json:"networks"`

	Ports string `json:"ports"`

	RunningFor string `json:"runningFor"`

	Size string `json:"size"`

	State string `json:"state"`

	Status string `json:"status"`

	SocketNumber string
}
