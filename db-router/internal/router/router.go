package router

// Target is one database the router can send a command to.
type Target struct {
	Name string `json:"name"` // e.g. "primary-9001"
	Addr string `json:"addr"` // e.g. "127.0.0.1:9001"
}

// Router is the whole point of this project: given a command, decide which database handles it.
type Router interface {
	Route(cmd Command) Target
}
